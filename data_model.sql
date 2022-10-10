-- MQTT authentication
CREATE EXTENSION pgcrypto;

CREATE TABLE vmq_auth_acl 
(
   mountpoint character varying(10) NOT NULL,
   client_id character varying(128) NOT NULL,
   username character varying(128) NOT NULL,
   password character varying(128),
   publish_acl json,
   subscribe_acl json,
   CONSTRAINT vmq_auth_acl_primary_key PRIMARY KEY (mountpoint, client_id, username)
);

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
