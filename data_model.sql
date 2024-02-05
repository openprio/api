-- MQTT authentication
CREATE EXTENSION pgcrypto;

CREATE TABLE vehicle_pre_registration
(
  data_owner_code character varying(50) NOT NULL,
  vehicle_number character varying(50) NOT NULL,
  token character varying(128),
  created_at timestamp NOT NULL,
  used_at timestamp,
  -- enforce that only one vehicle can be registered with the same data_owner_code, vehicle_number combination.
  PRIMARY KEY (data_owner_code, vehicle_number)
);
