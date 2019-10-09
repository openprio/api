package main

import(
	"net/http"
	"time"
	"github.com/patrickmn/go-cache"
	
	"openprio_api/rand"
)

type Passcode struct  {
	ExpireTime string `json:"expire_time"`
	Passcode string `json:"passcode"`
}

func (db *DB) GeneratePasscode(w http.ResponseWriter, r *http.Request) {
	randomNumber := rand.Number(6)
	expireTime := time.Now().Add(1 * time.Minute).Format(time.RFC3339)
	db.localCache.Set("passcode", randomNumber, cache.DefaultExpiration)
	result := Passcode{ExpireTime: expireTime, Passcode: randomNumber}
	rnd.JSON(w, http.StatusOK, result)
}