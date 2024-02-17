package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

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

func (app application) readPatterns(w http.ResponseWriter, r *http.Request) {
	patterns, err := database.DBReadRegistryPatterns(app.db)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error from database: %s", err.Error()), http.StatusInternalServerError)
		log.Print(err.Error())
		return
	}

	// Write JSON response
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "   ")
	if err := encoder.Encode(patterns); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %s", err.Error()), http.StatusInternalServerError)
		log.Print(err.Error())
		return
	}
}

func (app application) createPattern(w http.ResponseWriter, r *http.Request) {
	organizationId := r.Context().Value(organizationIDKey).(float64)
	inputPattern := registryModels.RegistryPattern{}
	if err := json.NewDecoder(r.Body).Decode(&inputPattern); err != nil {
		http.Error(w, fmt.Sprintf("Error decoding request: %s", err.Error()), http.StatusBadRequest)
		log.Print(err.Error())
		return
	}
	pettern, err := database.DBInsertRegistryPattern(app.db, inputPattern, int(organizationId))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error from database: %s", err.Error()), http.StatusInternalServerError)
		log.Print(err.Error())
		return
	}

	// Write JSON response
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "   ")
	if err := encoder.Encode(pettern); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %s", err.Error()), http.StatusInternalServerError)
		log.Print(err.Error())
		return
	}
}

func (app application) updatePattern(w http.ResponseWriter, r *http.Request) {
	organizationId := r.Context().Value(organizationIDKey).(float64)
	inputPattern := registryModels.Pattern{}
	if err := json.NewDecoder(r.Body).Decode(&inputPattern); err != nil {
		http.Error(w, fmt.Sprintf("Error decoding request: %s", err.Error()), http.StatusBadRequest)
		return
	}
	if inputPattern.Component != "" {
		http.Error(w, "Component may not be updated post-create", http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])
	presentPattern, err := database.DBReadRegistryPattern(app.db, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error from database: %s", err.Error()), http.StatusInternalServerError)
		log.Print(err.Error())
		return
	}
	if presentPattern.Owner != int(organizationId) {
		http.Error(w, "Unauthorized. May only update patterns owned by your organization", http.StatusForbidden)
		return
	}
	pattern, err := database.DBUpdateRegistryPattern(app.db, inputPattern, int(organizationId), id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error from database: %s", err.Error()), http.StatusInternalServerError)
		log.Print(err.Error())
		return
	}

	// Write JSON response
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "   ")
	if err := encoder.Encode(pattern); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %s", err.Error()), http.StatusInternalServerError)
		log.Print(err.Error())
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
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		userID, ok := claims[string(userIDKey)].(float64)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		organizationID, ok := claims[string(organizationIDKey)].(float64)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
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

	router := mux.NewRouter()
	autenticatedRouter := router.PathPrefix("/").Subrouter()
	autenticatedRouter.Use(app.authMiddleware)

	router.HandleFunc("/patterns", app.readPatterns).Methods("GET")
	// router.HandleFunc("/patterns/{id:[0-9]+}", app.readPattern).Methods("GET")
	autenticatedRouter.HandleFunc("/patterns", app.createPattern).Methods("POST")
	autenticatedRouter.HandleFunc("/patterns/{id:[0-9]+}", app.updatePattern).Methods("POST")
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", Loaded.Listen.Host, Loaded.Listen.Port), router))
}
