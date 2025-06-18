package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"openprio_api/rand"
)

func (db *DB) RegisterVehicle(w http.ResponseWriter, r *http.Request) {
	vehiclePreRegistration := VehiclePreRegistration{}
	if err := json.NewDecoder(r.Body).Decode(&vehiclePreRegistration); err != nil {
		result := ResponseError{Msg: "Invalid vehiclePreRegistration.", StatusCode: http.StatusBadRequest}
		rnd.JSON(w, http.StatusBadRequest, result)
		return
	}
	if vehiclePreRegistration.DataOwnerCode == "" {
		result := ResponseError{Msg: "Required field data_owner_code not specified.", StatusCode: http.StatusBadRequest}
		rnd.JSON(w, http.StatusBadRequest, result)
		return
	}
	if vehiclePreRegistration.VehicleNumber == "" {
		result := ResponseError{Msg: "Required field vehicle_number not specified.", StatusCode: http.StatusBadRequest}
		rnd.JSON(w, http.StatusBadRequest, result)
		return
	}
	if vehiclePreRegistration.Token == "" {
		result := ResponseError{Msg: "Required field token not specified.", StatusCode: http.StatusBadRequest}
		rnd.JSON(w, http.StatusBadRequest, result)
		return
	}

	res, err := db.checkToken(vehiclePreRegistration)
	if err != nil {
		result := ResponseError{Msg: err.Error(), StatusCode: http.StatusInternalServerError}
		rnd.JSON(w, http.StatusInternalServerError, result)
		return
	}
	if !res {
		result := ResponseError{Msg: "Invalid vehiclePreRegistration.", StatusCode: http.StatusForbidden}
		rnd.JSON(w, http.StatusForbidden, result)
		return
	}
	registration, err := db.registerVehicle(vehiclePreRegistration)
	if err != nil {
		result := ResponseError{Msg: err.Error(), StatusCode: http.StatusInternalServerError}
		rnd.JSON(w, http.StatusInternalServerError, result)
		return
	}
	db.setPreRegistrationUsed(vehiclePreRegistration)
	rnd.JSON(w, http.StatusOK, registration)
}

func (db *DB) checkToken(vehiclePreRegistration VehiclePreRegistration) (bool, error) {
	query := `
	WITH

	-- select either the data_owner_code, vehicle_number and token matching the data_owner_code, vehicle_number
	target_pre_registration as (
		SELECT data_owner_code, vehicle_number, token
		FROM (
		SELECT data_owner_code, vehicle_number, token 
		FROM vehicle_pre_registration 
		WHERE data_owner_code = $1 and vehicle_number = $2 and used_at IS NULL 
		UNION ALL
		SELECT null, null, gen_salt('bf')
		) vehicle_pre_registration
		limit 1 -- only return the first row, either the real data_owner_code, vehicle_number and token or the "null" one
	),
	
	-- perform bcrypt matching on the guaranteed single row from target_pre_registration
	valid_pre_registration as (
		SELECT data_owner_code, vehicle_number 
		FROM target_pre_registration 
		WHERE token = crypt($3, token)
	)
		
	-- Return true or false.
	SELECT
		CASE WHEN EXISTS 
		(
			SELECT * FROM vehicle_pre_registration NATURAL JOIN valid_pre_registration limit 1
		)
		THEN true
		ELSE false
	END
	`
	var result bool
	err := db.db.QueryRow(query, vehiclePreRegistration.DataOwnerCode, vehiclePreRegistration.VehicleNumber, vehiclePreRegistration.Token).Scan(&result)
	return result, err
}

func (db *DB) registerVehicle(vehicle VehiclePreRegistration) (*DeviceCredentials, error) {
	clientId := fmt.Sprintf("vehicle:%s:%s", vehicle.DataOwnerCode, vehicle.VehicleNumber)
	deviceCredentials := DeviceCredentials{ClientId: clientId, Username: clientId}
	deviceCredentials.Token = rand.String(50)

	err := db.deleteAccount(deviceCredentials.Username)
	if err != nil {
		log.Print("Something went wrong with deleting mqtt_uses ")
	}
	err = db.saveAccount(deviceCredentials.Username, deviceCredentials.Token)
	if err != nil {
		log.Print("Something went wrong with storing mqtt_user")
		return nil, err
	}

	err = db.saveAcl(deviceCredentials.Username, false)
	if err != nil {
		return nil, err
	}

	return &deviceCredentials, err
}

func (db *DB) setPreRegistrationUsed(vehicle VehiclePreRegistration) {
	query := `
	UPDATE vehicle_pre_registration 
	SET used_at = NOW()
	WHERE data_owner_code = $1 and vehicle_number = $2
	`
	db.db.Exec(query, vehicle.DataOwnerCode, vehicle.VehicleNumber)
}
