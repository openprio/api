package main

import (
	"fmt"
	"log"
	"os"
	"net/http"
	"time"
	"encoding/json"
	"database/sql"
	
	"openprio_api/rand"

	"github.com/gorilla/mux"
	"github.com/patrickmn/go-cache"
	_ "github.com/lib/pq"
)

type DB struct {
	localCache *cache.Cache
	db *sql.DB
}

func main() {
	db := DB{localCache: cache.New(1*time.Minute, 10*time.Minute) }
	connStr := os.Getenv("DB_URL")
	if connStr == "" {
		log.Fatal("No ENV variable DB_URL specified.")
	}
	var err error
	db.db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	
	router := mux.NewRouter()
	router.HandleFunc("/auth/generate_passcode", db.GeneratePasscode)
	router.HandleFunc("/auth/register_device", db.RegisterDevice).Methods("POST")

        log.Println("start server :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func (db *DB) GeneratePasscode(w http.ResponseWriter, r *http.Request) {
	log.Println("test")
        randomNumber := rand.Number(6)
        log.Println(randomNumber)
	db.localCache.Set("passcode", randomNumber, cache.DefaultExpiration)
	fmt.Fprintln(w, randomNumber)
}

type RegisterDevice struct  {
	DeviceId string `json:"device_id"`
	Passcode string `json:"passcode"`
}

type DeviceCredentials struct {
	ClientId string `json:"client_id"`
	Username string `json:"username"`
	Token string `json:"token"`
}

func (db *DB) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	device := RegisterDevice{}
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		log.Println(err)
	}

	passcode, found := db.localCache.Get("passcode")
	if found && passcode == device.Passcode {
		result, err := db.registerDeviceDB(device)
		if err != nil {
			log.Println(err)
			fmt.Fprintln(w, "Error")
			return
		}
		json.NewEncoder(w).Encode(result)
		return
	}
	fmt.Println(passcode)
	fmt.Println(device.Passcode)
	fmt.Fprintln(w, "Invalid or expired passcode")
}

func (db *DB) registerDeviceDB(device RegisterDevice) (*DeviceCredentials, error) {
	deviceCredentials := DeviceCredentials{ClientId: device.DeviceId, Username: device.DeviceId}
	deviceCredentials.Token = rand.String(32)

	query := `
	WITH x AS (
		SELECT
		    	''::text AS mountpoint,
			$1::text AS client_id,
		       	$2::text AS username,
		       	$3::text AS password,
		       	gen_salt('bf')::text AS salt,
		       	'[{"pattern": "/prod/pt/position/+/vehicle_number/+"}, {"pattern": "/test/pt/position/+/vehicle_number/+"}]'::json AS publish_acl,
		       	'[{"pattern": "/prod/pt/ssm/+/vehicle_number/+"}, {"pattern": "/test/pt/ssm/+/vehicle_number/+"}]'::json AS subscribe_acl
		) 
	INSERT INTO vmq_auth_acl (mountpoint, client_id, username, password, publish_acl, subscribe_acl)
	SELECT 
		x.mountpoint,
		x.client_id,
		x.username,
		crypt(x.password, x.salt),
		publish_acl,
		subscribe_acl
	FROM x;
	`
	_, err := db.db.Exec(query, deviceCredentials.ClientId, deviceCredentials.Username, deviceCredentials.Token)
	if err != nil {
		return nil, err
	}
	db.localCache.Delete("passcode")
	return &deviceCredentials, err
}