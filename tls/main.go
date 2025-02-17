package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jmpsec/osctrl/backend"
	"github.com/jmpsec/osctrl/cache"
	"github.com/jmpsec/osctrl/carves"
	"github.com/jmpsec/osctrl/environments"
	"github.com/jmpsec/osctrl/logging"
	"github.com/jmpsec/osctrl/metrics"
	"github.com/jmpsec/osctrl/nodes"
	"github.com/jmpsec/osctrl/queries"
	"github.com/jmpsec/osctrl/settings"
	"github.com/jmpsec/osctrl/tags"
	"github.com/jmpsec/osctrl/tls/handlers"
	"github.com/jmpsec/osctrl/types"
	"github.com/jmpsec/osctrl/version"
	"github.com/urfave/cli/v2"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

const (
	// Project name
	projectName string = "osctrl"
	// Service name
	serviceName string = projectName + "-" + settings.ServiceTLS
	// Service version
	serviceVersion string = version.OsctrlVersion
	// Service description
	serviceDescription string = "TLS service for osctrl"
	// Application description
	appDescription string = serviceDescription + ", a fast and efficient osquery management"
	// Default endpoint to handle HTTP health
	healthPath string = "/health"
	// Default endpoint to handle HTTP errors
	errorPath string = "/error"
	// Default service configuration file
	defConfigurationFile string = "config/" + settings.ServiceTLS + ".json"
	// Default DB configuration file
	defDBConfigurationFile string = "config/db.json"
	// Default redis configuration file
	defRedisConfigurationFile string = "config/redis.json"
	// Default Logger configuration file
	defLoggerConfigurationFile string = "config/logger.json"
	// Default always DB logger configuration file
	defAlwaysLogConfigurationFile string = "config/always.json"
	// Default carver configuration file
	defCarverConfigurationFile string = "config/carver.json"
	// Default TLS certificate file
	defTLSCertificateFile string = "config/tls.crt"
	// Default TLS private key file
	defTLSKeyFile string = "config/tls.key"
	// Default refreshing interval in seconds
	defaultRefresh int = 300
	// Default accelerate interval in seconds
	defaultAccelerate int = 60
	// Default expiration of oneliners for enroll/expire
	defaultOnelinerExpiration bool = true
)

var (
	// Wait for backend in seconds
	backendWait = 7 * time.Second
)

// Global variables
var (
	err             error
	tlsConfig       types.JSONConfigurationTLS
	dbConfig        backend.JSONConfigurationDB
	redisConfig     cache.JSONConfigurationRedis
	db              *backend.DBManager
	redis           *cache.RedisManager
	settingsmgr     *settings.Settings
	envs            *environments.Environment
	envsmap         environments.MapEnvironments
	settingsmap     settings.MapSettings
	nodesmgr        *nodes.NodeManager
	queriesmgr      *queries.Queries
	filecarves      *carves.Carves
	tlsMetrics      *metrics.Metrics
	ingestedMetrics *metrics.IngestedManager
	loggerTLS       *logging.LoggerTLS
	handlersTLS     *handlers.HandlersTLS
	tagsmgr         *tags.TagManager
	carvers3        *carves.CarverS3
	s3LogConfig     types.S3Configuration
	s3CarverConfig  types.S3Configuration
	app             *cli.App
	flags           []cli.Flag
)

// Variables for flags
var (
	configFlag        bool
	serviceConfigFile string
	redisConfigFile   string
	dbFlag            bool
	redisFlag         bool
	dbConfigFile      string
	tlsServer         bool
	tlsCertFile       string
	tlsKeyFile        string
	loggerFile        string
	alwaysLog         bool
	carverConfigFile  string
)

// Valid values for authentication in configuration
var validAuth = map[string]bool{
	settings.AuthNone: true,
}

// Valid values for logging in configuration
var validLogging = map[string]bool{
	settings.LoggingNone:    true,
	settings.LoggingStdout:  true,
	settings.LoggingFile:    true,
	settings.LoggingDB:      true,
	settings.LoggingGraylog: true,
	settings.LoggingSplunk:  true,
	settings.LoggingKafka:   true,
	settings.LoggingKinesis: true,
	settings.LoggingS3:      true,
}

// Valid values for carver in configuration
var validCarver = map[string]bool{
	settings.CarverDB:    true,
	settings.CarverLocal: true,
	settings.CarverS3:    true,
}

// Function to load the configuration file and assign to variables
func loadConfiguration(file, service string) (types.JSONConfigurationTLS, error) {
	var cfg types.JSONConfigurationTLS
	log.Printf("Loading %s", file)
	// Load file and read config
	viper.SetConfigFile(file)
	if err := viper.ReadInConfig(); err != nil {
		return cfg, err
	}
	// TLS endpoint values
	tlsRaw := viper.Sub(service)
	if tlsRaw == nil {
		return cfg, fmt.Errorf("JSON key %s not found in %s", service, file)
	}
	if err := tlsRaw.Unmarshal(&cfg); err != nil {
		return cfg, err
	}
	// Check if values are valid
	if !validAuth[cfg.Auth] {
		return cfg, fmt.Errorf("Invalid auth method")
	}
	if !validLogging[cfg.Logger] {
		return cfg, fmt.Errorf("Invalid logging method")
	}
	if !validCarver[cfg.Carver] {
		return cfg, fmt.Errorf("Invalid carver method")
	}
	// No errors!
	return cfg, nil
}

// Initialization code
func init() {
	// Initialize CLI flags
	flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "config",
			Aliases:     []string{"c"},
			Value:       false,
			Usage:       "Provide service configuration via JSON file",
			EnvVars:     []string{"SERVICE_CONFIG"},
			Destination: &configFlag,
		},
		&cli.StringFlag{
			Name:        "config-file",
			Aliases:     []string{"C"},
			Value:       defConfigurationFile,
			Usage:       "Load service configuration from `FILE`",
			EnvVars:     []string{"SERVICE_CONFIG_FILE"},
			Destination: &serviceConfigFile,
		},
		&cli.StringFlag{
			Name:        "listener",
			Aliases:     []string{"l"},
			Value:       "0.0.0.0",
			Usage:       "Listener for the service",
			EnvVars:     []string{"SERVICE_LISTENER"},
			Destination: &tlsConfig.Listener,
		},
		&cli.StringFlag{
			Name:        "port",
			Aliases:     []string{"p"},
			Value:       "9000",
			Usage:       "TCP port for the service",
			EnvVars:     []string{"SERVICE_PORT"},
			Destination: &tlsConfig.Port,
		},
		&cli.StringFlag{
			Name:        "auth",
			Aliases:     []string{"A"},
			Value:       settings.AuthNone,
			Usage:       "Authentication mechanism for the service",
			EnvVars:     []string{"SERVICE_AUTH"},
			Destination: &tlsConfig.Auth,
		},
		&cli.StringFlag{
			Name:        "host",
			Aliases:     []string{"H"},
			Value:       "0.0.0.0",
			Usage:       "Exposed hostname the service uses",
			EnvVars:     []string{"SERVICE_HOST"},
			Destination: &tlsConfig.Host,
		},
		&cli.StringFlag{
			Name:        "logger",
			Aliases:     []string{"L"},
			Value:       settings.LoggingDB,
			Usage:       "Logger mechanism to handle status/result logs from nodes",
			EnvVars:     []string{"SERVICE_LOGGER"},
			Destination: &tlsConfig.Logger,
		},
		&cli.BoolFlag{
			Name:        "redis",
			Aliases:     []string{"r"},
			Value:       false,
			Usage:       "Provide redis configuration via JSON file",
			EnvVars:     []string{"REDIS_CONFIG"},
			Destination: &redisFlag,
		},
		&cli.StringFlag{
			Name:        "redis-file",
			Aliases:     []string{"R"},
			Value:       defRedisConfigurationFile,
			Usage:       "Load redis configuration from `FILE`",
			EnvVars:     []string{"REDIS_CONFIG_FILE"},
			Destination: &redisConfigFile,
		},
		&cli.StringFlag{
			Name:        "redis-connection-string",
			Value:       "",
			Usage:       "Redis connection string, must include schema (<redis|rediss|unix>://<user>:<pass>@<host>:<port>/<db>?<options>",
			EnvVars:     []string{"REDIS_CONNECTION_STRING"},
			Destination: &redisConfig.ConnectionString,
		},
		&cli.StringFlag{
			Name:        "redis-host",
			Value:       "127.0.0.1",
			Usage:       "Redis host to be connected to",
			EnvVars:     []string{"REDIS_HOST"},
			Destination: &redisConfig.Host,
		},
		&cli.StringFlag{
			Name:        "redis-port",
			Value:       "6379",
			Usage:       "Redis port to be connected to",
			EnvVars:     []string{"REDIS_PORT"},
			Destination: &redisConfig.Port,
		},
		&cli.StringFlag{
			Name:        "redis-pass",
			Value:       "",
			Usage:       "Password to be used for redis",
			EnvVars:     []string{"REDIS_PASS"},
			Destination: &redisConfig.Password,
		},
		&cli.IntFlag{
			Name:        "redis-db",
			Value:       0,
			Usage:       "Redis database to be selected after connecting",
			EnvVars:     []string{"REDIS_DB"},
			Destination: &redisConfig.DB,
		},
		&cli.IntFlag{
			Name:        "redis-status-exp",
			Value:       cache.StatusExpiration,
			Usage:       "Redis expiration in hours for status logs",
			EnvVars:     []string{"REDIS_STATUS_EXP"},
			Destination: &redisConfig.StatusExpirationHours,
		},
		&cli.IntFlag{
			Name:        "redis-result-exp",
			Value:       cache.ResultExpiration,
			Usage:       "Redis expiration in hours for result logs",
			EnvVars:     []string{"REDIS_RESULT_EXP"},
			Destination: &redisConfig.ResultExpirationHours,
		},
		&cli.IntFlag{
			Name:        "redis-query-exp",
			Value:       cache.QueryExpiration,
			Usage:       "Redis expiration in hours for query logs",
			EnvVars:     []string{"REDIS_QUERY_EXP"},
			Destination: &redisConfig.QueryExpirationHours,
		},
		&cli.BoolFlag{
			Name:        "db",
			Aliases:     []string{"d"},
			Value:       false,
			Usage:       "Provide DB configuration via JSON file",
			EnvVars:     []string{"DB_CONFIG"},
			Destination: &dbFlag,
		},
		&cli.StringFlag{
			Name:        "db-file",
			Aliases:     []string{"D"},
			Value:       defDBConfigurationFile,
			Usage:       "Load DB configuration from `FILE`",
			EnvVars:     []string{"DB_CONFIG_FILE"},
			Destination: &dbConfigFile,
		},
		&cli.StringFlag{
			Name:        "db-host",
			Value:       "127.0.0.1",
			Usage:       "Backend host to be connected to",
			EnvVars:     []string{"DB_HOST"},
			Destination: &dbConfig.Host,
		},
		&cli.StringFlag{
			Name:        "db-port",
			Value:       "5432",
			Usage:       "Backend port to be connected to",
			EnvVars:     []string{"DB_PORT"},
			Destination: &dbConfig.Port,
		},
		&cli.StringFlag{
			Name:        "db-name",
			Value:       "osctrl",
			Usage:       "Database name to be used in the backend",
			EnvVars:     []string{"DB_NAME"},
			Destination: &dbConfig.Name,
		},
		&cli.StringFlag{
			Name:        "db-user",
			Value:       "postgres",
			Usage:       "Username to be used for the backend",
			EnvVars:     []string{"DB_USER"},
			Destination: &dbConfig.Username,
		},
		&cli.StringFlag{
			Name:        "db-pass",
			Value:       "postgres",
			Usage:       "Password to be used for the backend",
			EnvVars:     []string{"DB_PASS"},
			Destination: &dbConfig.Password,
		},
		&cli.IntFlag{
			Name:        "db-max-idle-conns",
			Value:       20,
			Usage:       "Maximum number of connections in the idle connection pool",
			EnvVars:     []string{"DB_MAX_IDLE_CONNS"},
			Destination: &dbConfig.MaxIdleConns,
		},
		&cli.IntFlag{
			Name:        "db-max-open-conns",
			Value:       100,
			Usage:       "Maximum number of open connections to the database",
			EnvVars:     []string{"DB_MAX_OPEN_CONNS"},
			Destination: &dbConfig.MaxOpenConns,
		},
		&cli.IntFlag{
			Name:        "db-conn-max-lifetime",
			Value:       30,
			Usage:       "Maximum amount of time a connection may be reused",
			EnvVars:     []string{"DB_CONN_MAX_LIFETIME"},
			Destination: &dbConfig.ConnMaxLifetime,
		},
		&cli.BoolFlag{
			Name:        "tls",
			Aliases:     []string{"t"},
			Value:       false,
			Usage:       "Enable TLS termination. It requires certificate and key",
			EnvVars:     []string{"TLS_SERVER"},
			Destination: &tlsServer,
		},
		&cli.StringFlag{
			Name:        "cert",
			Aliases:     []string{"T"},
			Value:       defTLSCertificateFile,
			Usage:       "TLS termination certificate from `FILE`",
			EnvVars:     []string{"TLS_CERTIFICATE"},
			Destination: &tlsCertFile,
		},
		&cli.StringFlag{
			Name:        "key",
			Aliases:     []string{"K"},
			Value:       defTLSKeyFile,
			Usage:       "TLS termination private key from `FILE`",
			EnvVars:     []string{"TLS_KEY"},
			Destination: &tlsKeyFile,
		},
		&cli.StringFlag{
			Name:        "logger-file",
			Aliases:     []string{"F"},
			Value:       defLoggerConfigurationFile,
			Usage:       "Logger configuration to handle status/results logs from nodes",
			EnvVars:     []string{"LOGGER_FILE"},
			Destination: &loggerFile,
		},
		&cli.BoolFlag{
			Name:        "always-log",
			Aliases:     []string{"a", "always"},
			Value:       false,
			Usage:       "Always log status and on-demand query logs from nodes in database",
			EnvVars:     []string{"ALWAYS_LOG"},
			Destination: &alwaysLog,
		},
		&cli.StringFlag{
			Name:        "carver-type",
			Value:       settings.CarverDB,
			Usage:       "Carver to be used to receive files extracted from nodes",
			EnvVars:     []string{"CARVER_TYPE"},
			Destination: &tlsConfig.Carver,
		},
		&cli.StringFlag{
			Name:        "carver-file",
			Value:       defCarverConfigurationFile,
			Usage:       "Carver configuration file to receive files extracted from nodes",
			EnvVars:     []string{"CARVER_FILE"},
			Destination: &carverConfigFile,
		},
		&cli.StringFlag{
			Name:        "log-s3-bucket",
			Value:       "",
			Usage:       "S3 bucket to be used as configuration for logging",
			EnvVars:     []string{"LOG_S3_BUCKET"},
			Destination: &s3LogConfig.Bucket,
		},
		&cli.StringFlag{
			Name:        "log-s3-region",
			Value:       "",
			Usage:       "S3 region to be used as configuration for logging",
			EnvVars:     []string{"LOG_S3_REGION"},
			Destination: &s3LogConfig.Region,
		},
		&cli.StringFlag{
			Name:        "log-s3-key-id",
			Value:       "",
			Usage:       "S3 access key id to be used as configuration for logging",
			EnvVars:     []string{"LOG_S3_KEY_ID"},
			Destination: &s3LogConfig.AccessKey,
		},
		&cli.StringFlag{
			Name:        "log-s3-secret",
			Value:       "",
			Usage:       "S3 access key secret to be used as configuration for logging",
			EnvVars:     []string{"LOG_S3_SECRET"},
			Destination: &s3LogConfig.SecretAccessKey,
		},
		&cli.StringFlag{
			Name:        "carver-s3-bucket",
			Value:       "",
			Usage:       "S3 bucket to be used as configuration for carves",
			EnvVars:     []string{"CARVER_S3_BUCKET"},
			Destination: &s3CarverConfig.Bucket,
		},
		&cli.StringFlag{
			Name:        "carver-s3-region",
			Value:       "",
			Usage:       "S3 region to be used as configuration for carves",
			EnvVars:     []string{"CARVER_S3_REGION"},
			Destination: &s3CarverConfig.Region,
		},
		&cli.StringFlag{
			Name:        "carve-s3-key-id",
			Value:       "",
			Usage:       "S3 access key id to be used as configuration for carves",
			EnvVars:     []string{"CARVER_S3_KEY_ID"},
			Destination: &s3CarverConfig.AccessKey,
		},
		&cli.StringFlag{
			Name:        "carve-s3-secret",
			Value:       "",
			Usage:       "S3 access key secret to be used as configuration for carves",
			EnvVars:     []string{"CARVER_S3_SECRET"},
			Destination: &s3CarverConfig.SecretAccessKey,
		},
	}
	// Logging format flags
	log.SetFlags(log.Lshortfile)
}

// Go go!
func osctrlService() {
	log.Println("Initializing backend...")
	// Attempt to connect to backend waiting until is ready
	for {
		db, err = backend.CreateDBManager(dbConfig)
		if db != nil {
			log.Println("Connection to backend successful!")
			break
		}
		if err != nil {
			log.Fatalf("Failed to connect to backend - %v", err)
		}
		log.Println("Backend NOT ready! waiting...")
		time.Sleep(backendWait)
	}
	log.Println("Initializing cache...")
	redis, err = cache.CreateRedisManager(redisConfig)
	if err != nil {
		log.Fatalf("Failed to connect to redis - %v", err)
	}
	log.Println("Connection to cache successful!")
	log.Println("Initialize environment")
	envs = environments.CreateEnvironment(db.Conn)
	log.Println("Initialize settings")
	settingsmgr = settings.NewSettings(db.Conn)
	log.Println("Initialize nodes")
	nodesmgr = nodes.CreateNodes(db.Conn)
	log.Println("Initialize tags")
	tagsmgr = tags.CreateTagManager(db.Conn)
	log.Println("Initialize queries")
	queriesmgr = queries.CreateQueries(db.Conn)
	log.Println("Initialize carves")
	filecarves = carves.CreateFileCarves(db.Conn, tlsConfig.Carver, carvers3)
	log.Println("Loading service settings")
	if err := loadingSettings(settingsmgr); err != nil {
		log.Fatalf("Error loading settings - %s: %v", tlsConfig.Logger, err)
	}
	// Initialize service metrics
	log.Println("Loading service metrics")
	tlsMetrics, err = loadingMetrics(settingsmgr)
	if err != nil {
		log.Fatalf("Error loading metrics - %v", err)
	}
	// Initialize ingested data metrics
	log.Println("Initialize ingested")
	ingestedMetrics = metrics.CreateIngested(db.Conn)
	// Initialize TLS logger
	log.Println("Loading TLS logger")
	loggerTLS, err = logging.CreateLoggerTLS(tlsConfig.Logger, loggerFile, s3LogConfig, alwaysLog, dbConfig, settingsmgr, nodesmgr, queriesmgr, redis)
	if err != nil {
		log.Fatalf("Error loading logger - %s: %v", tlsConfig.Logger, err)
	}

	// Sleep to reload environments
	// FIXME Implement Redis cache
	// FIXME splay this?
	log.Println("Preparing pseudo-cache for environments")
	go func() {
		_t := settingsmgr.RefreshEnvs(settings.ServiceTLS)
		if _t == 0 {
			_t = int64(defaultRefresh)
		}
		for {
			if settingsmgr.DebugService(settings.ServiceTLS) {
				log.Println("DebugService: Refreshing environments")
			}
			envsmap = refreshEnvironments()
			time.Sleep(time.Duration(_t) * time.Second)
		}
	}()
	// Sleep to reload settings
	// FIXME Implement Redis cache
	// FIXME splay this?
	log.Println("Preparing pseudo-cache for settings")
	go func() {
		_t := settingsmgr.RefreshSettings(settings.ServiceTLS)
		if _t == 0 {
			_t = int64(defaultRefresh)
		}
		for {
			if settingsmgr.DebugService(settings.ServiceTLS) {
				log.Println("DebugService: Refreshing settings")
			}
			settingsmap = refreshSettings()
			time.Sleep(time.Duration(_t) * time.Second)
		}
	}()
	// Initialize TLS handlers before router
	handlersTLS = handlers.CreateHandlersTLS(
		handlers.WithEnvs(envs),
		handlers.WithEnvsMap(&envsmap),
		handlers.WithNodes(nodesmgr),
		handlers.WithTags(tagsmgr),
		handlers.WithQueries(queriesmgr),
		handlers.WithCarves(filecarves),
		handlers.WithSettings(settingsmgr),
		handlers.WithSettingsMap(&settingsmap),
		handlers.WithMetrics(tlsMetrics),
		handlers.WithIngested(ingestedMetrics),
		handlers.WithLogs(loggerTLS),
	)

	// ///////////////////////// ALL CONTENT IS UNAUTHENTICATED FOR TLS
	if settingsmgr.DebugService(settings.ServiceTLS) {
		log.Println("DebugService: Creating router")
	}
	// Create router for TLS endpoint
	routerTLS := mux.NewRouter()
	// TLS: root
	routerTLS.HandleFunc("/", handlersTLS.RootHandler)
	// TLS: testing
	routerTLS.HandleFunc(healthPath, handlersTLS.HealthHandler).Methods("GET")
	// TLS: error
	routerTLS.HandleFunc(errorPath, handlersTLS.ErrorHandler).Methods("GET")
	// TLS: Specific routes for osquery nodes
	// FIXME this forces all paths to be the same
	routerTLS.HandleFunc("/{environment}/"+environments.DefaultEnrollPath, handlersTLS.EnrollHandler).Methods("POST")
	routerTLS.HandleFunc("/{environment}/"+environments.DefaultConfigPath, handlersTLS.ConfigHandler).Methods("POST")
	routerTLS.HandleFunc("/{environment}/"+environments.DefaultLogPath, handlersTLS.LogHandler).Methods("POST")
	routerTLS.HandleFunc("/{environment}/"+environments.DefaultQueryReadPath, handlersTLS.QueryReadHandler).Methods("POST")
	routerTLS.HandleFunc("/{environment}/"+environments.DefaultQueryWritePath, handlersTLS.QueryWriteHandler).Methods("POST")
	routerTLS.HandleFunc("/{environment}/"+environments.DefaultCarverInitPath, handlersTLS.CarveInitHandler).Methods("POST")
	routerTLS.HandleFunc("/{environment}/"+environments.DefaultCarverBlockPath, handlersTLS.CarveBlockHandler).Methods("POST")
	// TLS: Quick enroll/remove script
	routerTLS.HandleFunc("/{environment}/{secretpath}/{script}", handlersTLS.QuickEnrollHandler).Methods("GET")
	// TLS: osctrld retrieve flags
	routerTLS.HandleFunc("/{environment}/"+environments.DefaultFlagsPath, handlersTLS.FlagsHandler).Methods("POST")
	// TLS: osctrld retrieve certificate
	routerTLS.HandleFunc("/{environment}/"+environments.DefaultCertPath, handlersTLS.CertHandler).Methods("POST")
	// TLS: osctrld verification
	routerTLS.HandleFunc("/{environment}/"+environments.DefaultVerifyPath, handlersTLS.VerifyHandler).Methods("POST")
	// TLS: osctrld retrieve script to install/remove osquery
	routerTLS.HandleFunc("/{environment}/{action}/{platform}/"+environments.DefaultScriptPath, handlersTLS.ScriptHandler).Methods("POST")

	// ////////////////////////////// Everything is ready at this point!
	serviceListener := tlsConfig.Listener + ":" + tlsConfig.Port
	if tlsServer {
		log.Println("TLS Termination is enabled")
		cfg := &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		}
		srv := &http.Server{
			Addr:         serviceListener,
			Handler:      routerTLS,
			TLSConfig:    cfg,
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
		}
		log.Printf("%s v%s - HTTPS listening %s", serviceName, serviceVersion, serviceListener)
		log.Fatal(srv.ListenAndServeTLS(tlsCertFile, tlsKeyFile))
	} else {
		log.Printf("%s v%s - HTTP listening %s", serviceName, serviceVersion, serviceListener)
		log.Fatal(http.ListenAndServe(serviceListener, routerTLS))
	}
}

// Action to run when no flags are provided to run checks and prepare data
func cliAction(c *cli.Context) error {
	// Load configuration if external JSON config file is used
	if configFlag {
		tlsConfig, err = loadConfiguration(serviceConfigFile, settings.ServiceTLS)
		if err != nil {
			return fmt.Errorf("Error loading %s - %s", serviceConfigFile, err)
		}
	}
	// Load db configuration if external JSON config file is used
	if dbFlag {
		dbConfig, err = backend.LoadConfiguration(dbConfigFile, backend.DBKey)
		if err != nil {
			return fmt.Errorf("Failed to load DB configuration - %v", err)
		}
	}
	// Load redis configuration if external JSON config file is used
	if redisFlag {
		redisConfig, err = cache.LoadConfiguration(redisConfigFile, cache.RedisKey)
		if err != nil {
			return fmt.Errorf("Failed to load redis configuration - %v", err)
		}
	}
	// Load carver configuration if external JSON config file is used
	if tlsConfig.Carver == settings.CarverS3 {
		if s3CarverConfig.Bucket != "" {
			carvers3, err = carves.CreateCarverS3(s3CarverConfig)
		} else {
			carvers3, err = carves.CreateCarverS3File(carverConfigFile)
		}
		if err != nil {
			return fmt.Errorf("Failed to initiate s3 carver - %v", err)
		}
	}
	return nil
}

func main() {
	// Initiate CLI and parse arguments
	app = cli.NewApp()
	app.Name = serviceName
	app.Usage = appDescription
	app.Version = serviceVersion
	app.Description = appDescription
	app.Flags = flags
	// Define this command for help to exit when help flag is passed
	app.Commands = []*cli.Command{
		{
			Name: "help",
			Action: func(c *cli.Context) error {
				cli.ShowAppHelpAndExit(c, 0)
				return nil
			},
		},
	}
	app.Action = cliAction
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
	// Service starts!
	osctrlService()
}
