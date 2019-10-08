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

CREATE TABLE devices_registered
(
  device_id character varying(128) NOT NULL,
  registered_at timestamp NOT NULL,
  public_transport_agency varying(255) NOT NULL,
  PRIMARY KEY(device_id)
);
