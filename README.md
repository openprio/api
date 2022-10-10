# OpenPrio authentication

OpenPrio uses MQTT to communicate between the vehicle and a central server that distributes the messages to third parties. It's important that the central server only distributes messages from real authenticated vehicles, since one of the usecases of OpenPrio is granting priority to Public Transport. This means authentication and authorization is required. This document describes how the authentication and authorization process works. 

## General introduction

In OpenPrio a vehicle is uniquely identified by the data_owner_code (describes an operator uniquely) and the vehicle_number (a unique number for a vehicle within an operator). Every vehicle gets exactly one credential to create one connection with the MQTT broker, if a second connection is opened with the same credentials the first connection will be terminated. The authorization and authentication system is designed with the usage of a OBU (on-board unit ) in mind. The process starts with the distribution of pre-registrations for vehicles to operators, the operator should make sure that each vehicle gets the right pre-registration. The OBU of the vehicle is responsible for exchanging the pre-registration for a permanent registration and store this registration locally. The OBU of the vehicle is the only place where this credentials should be stored.

### Flow
![image description](docs/images/authentication_flow.png)

1. Pre-registering vehicles
The operator communicates to the OpenPrio administrator the wish to register vehicles in a certain range, for example: 3000-3049,3052,3060-3080,4001-4072,5001-5070. The operator receives a JSON and/or .csv containing  the pre-registrations consisting of data_owner_code, vehicle_number and token, see example below.

```json
[
    {
        "data_owner_code": "HTM",
        "vehicle_number": "5001",
        "token": "<token>",
        "created_at": "2022-10-10T11:14:06+02:00"
    },
    {
        "data_owner_code": "HTM",
        "vehicle_number": "5002",
        "token": "<token>",
        "created_at": "2022-10-10T11:14:06+02:00"
    },
    {
        "data_owner_code": "HTM",
        "vehicle_number": "5070",
        "token": "<token>",
        "created_at": "2022-10-10T11:14:06+02:00"
    }
]
```

2. Distributing pre-registrations to vehicles

The operator should distribute the pre-registrations to the OBU's of the vehicles. This can be done manually, but preferably is this automated. 

3. Exchanging pre-registration for registration

After receiving a new pre-registration in the vehicle OBY this should be exchanged for final registration by making an HTTP-call to the the /register_vehicle endpoint.

```bash
curl --location --request POST 'https://api.openprio.nl/register_vehicle' \
--header 'Content-Type: application/json' \
--data-raw '{
        "data_owner_code": "HTM",
        "vehicle_number": "5070",
        "token": "<token>"
    }'
```
In return the OBU receives a client_id, username and token needed to connect MQTT, this credentials should be stored permanently on the OBU.

```json
{
    "client_id": "<client_id>",
    "username": "<username>",
    "token": "<token>"
}
```

4. Connecting with MQTT-broker

With the aquired credentials you can connect with the OpenPrio MQTT broker, the MQTT broker is reachable on mqtt.openprio.nl:8883. The following topics can be used: 

For publishing: 
To send position data according to the OpenPrio specification, https://github.com/openprio/specification/blob/master/openprio_pt_position_data.proto:

```
/prod/pt/position/<data_owner_code>/vehicle_number/<vehicle_number>
/test/pt/position/<data_owner_code>/vehicle_number/<vehicle_number>
```
For subscribing:
To receive feedback from the Traffic Light Controller, data according to the Extended SSM specification, https://github.com/openprio/specification/blob/master/ssm.proto
```
/prod/pt/ssm/<data_owner_code>/vehicle_number/<vehicle_number>
/test/pt/ssm/<data_owner_code>/vehicle_number/<vehicle_number>
```


