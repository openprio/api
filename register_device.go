package main

import (
	"encoding/json"
	"log"
	"net/http"

	"openprio_api/rand"
)

type RegisterDevice struct {
	DeviceId string `json:"device_id"`
	Passcode string `json:"passcode"`
}

type DeviceCredentials struct {
	ClientId string `json:"client_id"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

type ResponseError struct {
	Msg        string `json:"error_message"`
	StatusCode int    `json:"status_code"`
}

func (db *DB) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	device := RegisterDevice{}
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		log.Println(err)
	}
	if device.DeviceId == "" {
		result := ResponseError{Msg: "Required field device_id not specified.", StatusCode: http.StatusBadRequest}
		rnd.JSON(w, http.StatusBadRequest, result)
		return
	}

	passcode, found := db.localCache.Get("passcode")
	if found && passcode == device.Passcode {
		result, err := db.registerDeviceDB(device)
		if err != nil {
			log.Println(err)
			result := ResponseError{Msg: "Something went wrong check server log.", StatusCode: http.StatusInternalServerError}
			rnd.JSON(w, http.StatusInternalServerError, result)
			return
		}
		rnd.JSON(w, http.StatusOK, result)
		return
	}
	result := ResponseError{Msg: "Invalid or expired passcode.", StatusCode: http.StatusForbidden}
	rnd.JSON(w, http.StatusForbidden, result)
}

func (db *DB) saveAccount(username string, token string) error {
	query := `WITH x AS (
		SELECT
		    $1::text AS username,
		    $2::text AS password,
		    gen_salt('bf')::text AS salt
		) 
	INSERT INTO mqtt_user (username, password_hash)
	SELECT 
		x.username,
		crypt(x.password, x.salt)
	FROM x;
	`
	_, err := db.db.Exec(query, username, token)
	return err
}

func (db *DB) saveAcl(username string) error {
	query := `INSERT INTO mqtt_acl (username, clientid, action, permission, topic)
	VALUES ($1, $1, $2, 'allow', $3)
	`
	_, err := db.db.Exec(query, username, "subscribe", "/prod/pt/ssm/+/vehicle_number/+")
	if err != nil {
		return err
	}
	db.db.Exec(query, username, "subscribe", "/test/pt/ssm/+/vehicle_number/+")
	db.db.Exec(query, username, "publish", "/prod/pt/position/+/vehicle_number/+")
	db.db.Exec(query, username, "publish", "/test/pt/position/+/vehicle_number/+")
	return nil
}

func (db *DB) registerDeviceDB(device RegisterDevice) (*DeviceCredentials, error) {
	deviceCredentials := DeviceCredentials{ClientId: device.DeviceId, Username: device.DeviceId}
	deviceCredentials.Token = rand.String(32)

	err := db.saveAccount(deviceCredentials.Username, deviceCredentials.Token)
	if err != nil {
		log.Print("Something went wrong with storing mqtt_user")
		return nil, err
	}

	err = db.saveAcl(deviceCredentials.Username)
	if err != nil {
		return nil, err
	}
	db.localCache.Delete("passcode")
	return &deviceCredentials, err
}
