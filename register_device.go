package main

import(
	"encoding/json"
	"net/http"
	"log"
	
	"openprio_api/rand"
)

type RegisterDevice struct  {
	DeviceId string `json:"device_id"`
	Passcode string `json:"passcode"`
}

type DeviceCredentials struct {
	ClientId string `json:"client_id"`
	Username string `json:"username"`
	Token string `json:"token"`
}

type ResponseError struct {
	Msg string `json:"error_message"`
	StatusCode int `json:"status_code"`
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

func (db *DB) registerDeviceDB(device RegisterDevice) (*DeviceCredentials, error) {
	deviceCredentials := DeviceCredentials{ClientId: device.DeviceId, Username: device.DeviceId}
	deviceCredentials.Token = rand.String(32)

	// In the future the public_acl and subscribe_acl can be further limited by registering vehicle_number and data_owner code at registration.
	// Because an onboard computer is normally not swapped between buses. 
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