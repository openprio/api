package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"openprio_api/rand"
)

type VehiclePreRegistration struct {
	DataOwnerCode string `json:"data_owner_code"`
	VehicleNumber string `json:"vehicle_number"`
	Token         string `json:"token,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
}

func (db *DB) GenerateVehiclePreRegistrations(w http.ResponseWriter, r *http.Request) {
	var vehiclePreRegistrations []VehiclePreRegistration
	err := json.NewDecoder(r.Body).Decode(&vehiclePreRegistrations)
	if err != nil {
		result := ResponseError{Msg: err.Error(), StatusCode: http.StatusBadRequest}
		rnd.JSON(w, http.StatusBadRequest, result)
		return
	}

	registeredVehicles := map[string]bool{}
	var filteredVehiclePreRegistrations []VehiclePreRegistration
	// Check all to be registered vehicleRegistrations
	for _, vehiclePreRegistration := range vehiclePreRegistrations {
		key := fmt.Sprintf("%s:%s", vehiclePreRegistration.DataOwnerCode, vehiclePreRegistration.VehicleNumber)
		// skip double registration requests.
		if _, exists := registeredVehicles[key]; !exists {
			responseError := db.checkToPreRegisterVehicle(vehiclePreRegistration)
			if responseError.Msg != "" {
				rnd.JSON(w, responseError.StatusCode, responseError)
				return
			}
			filteredVehiclePreRegistrations = append(filteredVehiclePreRegistrations, vehiclePreRegistration)
			registeredVehicles[key] = true
		}
	}
	var vehiclePreRegistrationsResults []VehiclePreRegistration
	// Preregister
	for _, vehicleRegistration := range filteredVehiclePreRegistrations {
		result, responseError := db.preRegisterVehicle(vehicleRegistration)
		if responseError.Msg != "" {
			rnd.JSON(w, responseError.StatusCode, responseError)
			return
		}
		vehiclePreRegistrationsResults = append(vehiclePreRegistrationsResults, result)
	}

	rnd.JSON(w, http.StatusOK, vehiclePreRegistrationsResults)
}

func (db *DB) checkToPreRegisterVehicle(vehiclePreRegistration VehiclePreRegistration) ResponseError {
	exists, err := db.checkIfVehicleAlreadyRegistered(vehiclePreRegistration)
	if err != nil {
		return ResponseError{Msg: err.Error(), StatusCode: http.StatusInternalServerError}

	}
	if exists {
		msg := fmt.Sprintf("preregistration for data_owner_code: %s, vehicle_number: %s already exists.", vehiclePreRegistration.DataOwnerCode, vehiclePreRegistration.VehicleNumber)
		return ResponseError{Msg: msg, StatusCode: http.StatusBadRequest}
	}
	return ResponseError{}
}

func (db *DB) preRegisterVehicle(vehiclePreRegistration VehiclePreRegistration) (VehiclePreRegistration, ResponseError) {
	result, err := db.preRegisterVehicleInDB(vehiclePreRegistration)
	if err != nil {
		return vehiclePreRegistration, ResponseError{Msg: err.Error(), StatusCode: http.StatusInternalServerError}
	}
	return result, ResponseError{}
}

func (db *DB) checkIfVehicleAlreadyRegistered(vehiclePreRegistration VehiclePreRegistration) (bool, error) {
	query := `
		SELECT
			CASE WHEN EXISTS 
			(
				SELECT * FROM vehicle_pre_registration where data_owner_code = $1 and vehicle_number = $2
			)
			THEN true
    		ELSE false
		END
	`
	var result bool
	err := db.db.QueryRow(query, vehiclePreRegistration.DataOwnerCode, vehiclePreRegistration.VehicleNumber).Scan(&result)
	return result, err
}

func (db *DB) preRegisterVehicleInDB(vehiclePreRegistration VehiclePreRegistration) (VehiclePreRegistration, error) {
	vehiclePreRegistration.Token = rand.String(50)
	vehiclePreRegistration.CreatedAt = time.Now().Format(time.RFC3339)

	query := `
	WITH x AS (
		SELECT
				$1::text AS data_owner_code,
				$2::text AS vehicle_number,
				$3::text AS token,
				gen_salt('bf')::text AS salt,
				$4::timestamp AS created_at
		)
	INSERT INTO vehicle_pre_registration (data_owner_code, vehicle_number, token, created_at)
	SELECT
		data_owner_code,
		vehicle_number,
		crypt(x.token, x.salt),
		x.created_at
	FROM x;
	`
	_, err := db.db.Exec(query, vehiclePreRegistration.DataOwnerCode, vehiclePreRegistration.VehicleNumber,
		vehiclePreRegistration.Token, vehiclePreRegistration.CreatedAt)
	if err != nil {
		return VehiclePreRegistration{}, err
	}
	return vehiclePreRegistration, err
}
