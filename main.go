package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/patrickmn/go-cache"
	"github.com/thedevsaddam/renderer"
)

var rnd *renderer.Render

type DB struct {
	localCache *cache.Cache
	db         *sql.DB
}

func main() {
	db := DB{localCache: cache.New(1*time.Minute, 10*time.Minute)}
	connStr := os.Getenv("DB_URL")
	if connStr == "" {
		log.Fatal("No ENV variable DB_URL specified.")
	}
	var err error
	db.db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	rnd = renderer.New()

	router := mux.NewRouter()
	router.HandleFunc("/auth/generate_passcode", db.GeneratePasscode)
	router.HandleFunc("/register_device", db.RegisterDevice).Methods("POST")
	router.HandleFunc("/auth/generate_vehicle_pre_registrations", db.GenerateVehiclePreRegistrations).Methods("POST")
	router.HandleFunc("/register_vehicle", db.RegisterVehicle).Methods("POST")

	log.Println("start server :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
