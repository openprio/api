package main

import (
	"encoding/json"
	"fmt"
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

func (db *DB) registerVehicle(vehicle VehiclePreRegistration) (DeviceCredentials, error) {
	clientId := fmt.Sprintf("vehicle:%s:%s", vehicle.DataOwnerCode, vehicle.VehicleNumber)
	deviceCredentials := DeviceCredentials{ClientId: clientId, Username: clientId}
	deviceCredentials.Token = rand.String(50)

	query := `
	WITH x AS (
		SELECT
		    	''::text AS mountpoint,
				$1::text AS client_id,
		       	$2::text AS username,
		       	$3::text AS password,
		       	gen_salt('bf')::text AS salt,
		       	$4::json AS publish_acl,
		       	$5::json AS subscribe_acl
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
	positionProdTopic := fmt.Sprintf(`[
		{"pattern": "/prod/pt/position/%[1]s/vehicle_number/%[2]s"},
		{"pattern": "/test/pt/position/%[1]s/vehicle_number/%[2]s"}
	]`, vehicle.DataOwnerCode, vehicle.VehicleNumber)
	ssmProdTopic := fmt.Sprintf(`[
		{"pattern": "/prod/pt/ssm/%[1]s/vehicle_number/%[2]s"},
		{"pattern": "/test/pt/ssm/%[1]s/vehicle_number/%[2]s"}
	]`, vehicle.DataOwnerCode, vehicle.VehicleNumber)

	_, err := db.db.Exec(query, deviceCredentials.ClientId, deviceCredentials.Username, deviceCredentials.Token, positionProdTopic, ssmProdTopic)
	if err != nil {
		return DeviceCredentials{}, err
	}
	return deviceCredentials, err
}

func (db *DB) setPreRegistrationUsed(vehicle VehiclePreRegistration) {
	query := `
	UPDATE vehicle_pre_registration 
	SET used_at = NOW()
	WHERE data_owner_code = $1 and vehicle_number = $2
	`
	db.db.Exec(query, vehicle.DataOwnerCode, vehicle.VehicleNumber)
}
