package env

import (
	"flag"
	"fmt"
	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
	"log"
	"os"
)

// ErrDSNNotDefine in dsn database not defines from variables
var ErrDSNNotDefine = "DSN not define"

// ErrServerAddressNotDefined if address not defines
var ErrServerAddressNotDefined = "Server address not defined"

// Env project environment configuration
type Env struct {
	DatabaseDsn          string `env:"DATABASE_URI" envDefault:""`
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS" envDefault:""`
	ServerAddress        string `env:"RUN_ADDRESS" envDefault:""`
}

// Constants for variables
const (
	DatabaseDsn          = "DATABASE_URI"
	AccrualSystemAddress = "ACCRUAL_SYSTEM_ADDRESS"
	ServerAddress        = "RUN_ADDRESS"
)

// Maps for take inv params
var varToInv = map[string]string{
	DatabaseDsn:          "d",
	AccrualSystemAddress: "r",
	ServerAddress:        "a",
}

// New construct for Env struct
func New() (e *Env) {
	e = new(Env)
	e.init()

	return
}

// Init env params for project
func (e *Env) init() {
	// reset from env vars
	_ = env.Parse(e)
	//reset from flag
	dsn := flag.String(varToInv[DatabaseDsn], "", "")
	accrualAddr := flag.String(varToInv[AccrualSystemAddress], "", "")
	serverAddr := flag.String(varToInv[ServerAddress], "", "")
	flag.Parse()

	e.DatabaseDsn = *dsn
	if e.DatabaseDsn == "" {
		e.DatabaseDsn = e.fromDotEnv(DatabaseDsn)
	}

	e.AccrualSystemAddress = *accrualAddr
	if e.AccrualSystemAddress == "" {
		e.AccrualSystemAddress = e.fromDotEnv(AccrualSystemAddress)
	}

	e.ServerAddress = *serverAddr
	if e.ServerAddress == "" {
		e.ServerAddress = e.fromDotEnv(ServerAddress)
	}

	if e.DatabaseDsn == "" {
		log.Fatal(ErrDSNNotDefine)
	}
	e.ServerAddress = e.fromDotEnv(ServerAddress)
	if e.ServerAddress == "" {
		log.Fatal(ErrServerAddressNotDefined)
	}
}

// use godot package to load/read the .env file and
// return the value of the key
func (e *Env) fromDotEnv(key string) string {
	err := godotenv.Load("./configs/.env")
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return os.Getenv(key)
}
