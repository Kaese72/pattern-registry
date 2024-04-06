package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/Kaese72/pattern-registry/apierrors"
	"github.com/Kaese72/pattern-registry/internal/database"

	registryModels "github.com/Kaese72/pattern-registry/registry/models"
	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

type application struct {
	db        *sql.DB
	jwtSecret string
}

func terminalHTTPError(w http.ResponseWriter, err error) {
	var apiError apierrors.APIError
	if errors.As(err, &apiError) {
		if apiError.Code == 500 {
			// When an unknown error occurs, do not send the error to the client
			http.Error(w, "Internal Server Error", apiError.Code)
			log.Print(err.Error())
			return

		} else {
			bytes, intErr := json.MarshalIndent(apiError, "", "   ")
			if intErr != nil {
				// Must send a normal Error an not APIError just in case of eternal loop
				terminalHTTPError(w, fmt.Errorf("error encoding response: %s", intErr.Error()))
				return
			}
			http.Error(w, string(bytes), apiError.Code)
			return
		}
	} else {
		terminalHTTPError(w, apierrors.APIError{Code: http.StatusInternalServerError, WrappedError: err})
		return
	}
}

var queryRegex = regexp.MustCompile(`^(?P<key>\w+)\[(?P<operator>\w+)\]$`)

func parseQueryFilters(r *http.Request) []database.Filter {
	filters := []database.Filter{}
	for key, values := range r.URL.Query() {
		matches := queryRegex.FindStringSubmatch(key)
		if len(matches) == 0 {
			continue
		}
		for _, value := range values {
			filters = append(filters, database.Filter{Key: matches[queryRegex.SubexpIndex("key")], Value: value, Operator: matches[queryRegex.SubexpIndex("operator")]})
		}
	}
	return filters
}

func (app application) readPatterns(w http.ResponseWriter, r *http.Request) {
	patterns, err := database.DBReadRegistryPatterns(app.db, parseQueryFilters(r))
	if err != nil {
		terminalHTTPError(w, fmt.Errorf("error from database: %s", err.Error()))
		return
	}

	// Write JSON response
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "   ")
	if err := encoder.Encode(patterns); err != nil {
		terminalHTTPError(w, fmt.Errorf("error encoding response: %s", err.Error()))
		return
	}
}

func (app application) createPattern(w http.ResponseWriter, r *http.Request) {
	organizationId := r.Context().Value(organizationIDKey).(float64)
	inputPattern := registryModels.RegistryPattern{}
	if err := json.NewDecoder(r.Body).Decode(&inputPattern); err != nil {
		terminalHTTPError(w, fmt.Errorf("error decoding request: %s", err.Error()))
		return
	}
	pettern, err := database.DBInsertRegistryPattern(app.db, inputPattern, int(organizationId))
	if err != nil {
		terminalHTTPError(w, fmt.Errorf("error from database: %s", err.Error()))
		return
	}

	// Write JSON response
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "   ")
	if err := encoder.Encode(pettern); err != nil {
		terminalHTTPError(w, fmt.Errorf("error encoding response: %s", err.Error()))
		return
	}
}

func (app application) updatePattern(w http.ResponseWriter, r *http.Request) {
	organizationId := r.Context().Value(organizationIDKey).(float64)
	inputPattern := registryModels.Pattern{}
	if err := json.NewDecoder(r.Body).Decode(&inputPattern); err != nil {
		terminalHTTPError(w, fmt.Errorf("error decoding request: %s", err.Error()))
		return
	}
	if inputPattern.Component != "" {
		terminalHTTPError(w, fmt.Errorf("component may not be updated post-create"))
		return
	}

	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"]) // Ignoring error because mux guarantees this is an int
	presentPattern, err := database.DBReadRegistryPattern(app.db, id)
	if err != nil {
		terminalHTTPError(w, fmt.Errorf("error from database: %s", err.Error()))
		return
	}
	if presentPattern.Owner != int(organizationId) {
		terminalHTTPError(w, fmt.Errorf("unauthorized. May only update patterns owned by your organization"))
		return
	}
	pattern, err := database.DBUpdateRegistryPattern(app.db, inputPattern, int(organizationId), id)
	if err != nil {
		terminalHTTPError(w, fmt.Errorf("error from database: %s", err.Error()))
		return
	}

	// Write JSON response
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "   ")
	if err := encoder.Encode(pattern); err != nil {
		terminalHTTPError(w, fmt.Errorf("error encoding response: %s", err.Error()))
		return
	}
}

func (app application) deletePattern(w http.ResponseWriter, r *http.Request) {
	organizationId := r.Context().Value(organizationIDKey).(float64)
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"]) // Ignoring error because mux guarantees this is an int
	presentPattern, err := database.DBReadRegistryPattern(app.db, id)
	if err != nil {
		terminalHTTPError(w, fmt.Errorf("error from database: %s", err.Error()))
		return
	}
	if presentPattern.Owner != int(organizationId) {
		terminalHTTPError(w, fmt.Errorf("unauthorized. May only delete patterns owned by your organization"))
		return
	}
	err = database.DBDeleteRegistryPattern(app.db, id)
	if err != nil {
		terminalHTTPError(w, fmt.Errorf("error from database: %s", err.Error()))
		return
	}
}

type contextKey string

const (
	userIDKey         contextKey = "userID"
	organizationIDKey contextKey = "organizationID"
)

func (app application) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		tokenString = strings.Replace(tokenString, "Bearer ", "", 1)

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(app.jwtSecret), nil
		})

		if err != nil {
			terminalHTTPError(w, apierrors.APIError{Code: http.StatusUnauthorized, WrappedError: fmt.Errorf("error parsing token: %s", err.Error())})
			return
		}

		if !token.Valid {
			terminalHTTPError(w, apierrors.APIError{Code: http.StatusUnauthorized, WrappedError: errors.New("invalid token")})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			terminalHTTPError(w, apierrors.APIError{Code: http.StatusUnauthorized, WrappedError: errors.New("could not read claims")})
			return
		}

		userID, ok := claims[string(userIDKey)].(float64)
		if !ok {
			terminalHTTPError(w, apierrors.APIError{Code: http.StatusUnauthorized, WrappedError: errors.New("could not read userId claim")})
			return
		}
		organizationID, ok := claims[string(organizationIDKey)].(float64)
		if !ok {
			terminalHTTPError(w, apierrors.APIError{Code: http.StatusUnauthorized, WrappedError: errors.New("could not read organizationId claim")})
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		ctx = context.WithValue(ctx, organizationIDKey, organizationID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type Config struct {
	Database struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		Database string `mapstructure:"database"`
	} `mapstructure:"database"`
	JWT struct {
		Secret string `mapstructure:"secret"`
	} `mapstructure:"jwt"`
	Listen struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"listen"`
}

var Loaded Config

func init() {
	// Load configuration from environment
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.BindEnv("database.host")
	viper.BindEnv("database.port")
	viper.SetDefault("database.port", "3306")
	viper.BindEnv("database.user")
	viper.BindEnv("database.password")
	viper.BindEnv("database.database")
	viper.SetDefault("database.database", "patternregistry")

	// JWT configuration
	viper.BindEnv("jwt.secret")

	// HTTP listen config
	viper.BindEnv("listen.host")
	viper.SetDefault("listen.host", "0.0.0.0")
	viper.BindEnv("listen.port")
	viper.SetDefault("listen.port", "8080")

	err := viper.Unmarshal(&Loaded)
	if err != nil {
		log.Fatal(err.Error())
	}

	if Loaded.Database.Host == "" {
		log.Fatal("Database host not set")
	}

	if Loaded.Database.Password == "" {
		log.Fatal("Database password not set")
	}

	if Loaded.Database.User == "" {
		log.Fatal("Database user not set")
	}

	if Loaded.JWT.Secret == "" {
		log.Fatal("JWT secret key not set")
	}
}

func main() {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", Loaded.Database.User, Loaded.Database.Password, Loaded.Database.Host, Loaded.Database.Port, Loaded.Database.Database))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := application{
		db:        db,
		jwtSecret: Loaded.JWT.Secret,
	}

	// pattern-registry is the API identifier for this API
	router := mux.NewRouter().PathPrefix("/pattern-registry").Subrouter()
	autenticatedRouter := router.PathPrefix("/").Subrouter()
	autenticatedRouter.Use(app.authMiddleware)

	router.HandleFunc("/patterns", app.readPatterns).Methods("GET")
	// router.HandleFunc("/patterns/{id:[0-9]+}", app.readPattern).Methods("GET")
	autenticatedRouter.HandleFunc("/patterns", app.createPattern).Methods("POST")
	autenticatedRouter.HandleFunc("/patterns/{id:[0-9]+}", app.updatePattern).Methods("POST")
	autenticatedRouter.HandleFunc("/patterns/{id:[0-9]+}", app.deletePattern).Methods("DELETE")
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", Loaded.Listen.Host, Loaded.Listen.Port), router))
}
