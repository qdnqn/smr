package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/dgraph-io/badger/v4"
	"github.com/gin-gonic/gin"
	mdns "github.com/miekg/dns"
	_ "github.com/simplecontainer/smr/docs"
	"github.com/simplecontainer/smr/pkg/api"
	"github.com/simplecontainer/smr/pkg/commands"
	_ "github.com/simplecontainer/smr/pkg/commands"
	"github.com/simplecontainer/smr/pkg/configuration"
	"github.com/simplecontainer/smr/pkg/helpers"
	"github.com/simplecontainer/smr/pkg/logger"
	"github.com/simplecontainer/smr/pkg/mtls"
	"github.com/simplecontainer/smr/pkg/plugins"
	"github.com/simplecontainer/smr/pkg/startup"
	"github.com/simplecontainer/smr/pkg/static"
	"github.com/spf13/viper"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"net/http"
	"os"
	"strconv"
)

//	@title			Simple container manager API
//	@version		1.0
//	@description	This is a container orchestrator service.
//	@termsOfService	http://smr.qdnqn.com/terms

//	@contact.name	API Support
//	@contact.url	https://github.com/simplecontainer/smr

//	@license.name	GNU General Public License v3.0
//	@license.url	https://github.com/simplecontainer/smr/blob/main/LICENSE

//	@host		localhost:8080
//	@BasePath	/api/v1

//	@securityDefinitions.basic	BasicAuth

//	@externalDocs.description	OpenAPI
//	@externalDocs.url			https://swagger.io/resources/open-api/

func main() {
	startup.SetFlags()
	logger.Log = logger.NewLogger()

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = static.DEFAULT_LOG_LEVEL
	}

	fmt.Println(fmt.Sprintf("logging level set to %s (override with LOG_LEVEL env variable)", logLevel))

	// Prepare configuration for the commands
	conf := configuration.NewConfig()
	conf.Environment = startup.GetEnvironmentInfo()
	startup.ReadFlags(conf)

	var db *badger.DB
	api := api.NewApi(conf, db)
	api.Manager.LogLevel = helpers.GetLogLevel(logLevel)

	// Run any commands before starting daemon
	commands.PreloadCommands()
	commands.Run(api.Manager)

	configFile, err := os.Open(fmt.Sprintf("%s/%s/config.yaml", conf.Environment.PROJECTDIR, static.CONFIGDIR))

	if err != nil {
		panic(err)
	}

	conf, err = startup.Load(configFile)

	if err != nil {
		panic(err)
	}

	// Prepare configuration for the daeamon
	api.Config = conf
	api.Config.Environment = startup.GetEnvironmentInfo()
	startup.ReadFlags(api.Config)

	api.Keys = mtls.NewKeys("/home/smr-agent/.ssh")
	api.Manager.Keys = api.Keys

	fmt.Println(api.Config)

	var found bool
	found, err = mtls.GenerateIfNoKeysFound(api.Keys, api.Config)

	if err != nil {
		panic(err)
	}

	if !found {
		err = mtls.SaveToDirectory(api.Keys)

		if err != nil {
			logger.Log.Error("failed to save keys to directory", zap.String("error", err.Error()))
			os.Exit(1)
		}

		fmt.Println("Certificate is generated for the use by the smr client!")
		fmt.Println("Copy-paste it to safe location for further use - it will not be printed anymore in the logs")
		fmt.Println(mtls.GeneratePemBundle(api.Keys))
		os.Exit(0)
	}

	if viper.GetBool("daemon") {
		mdns.HandleFunc(".", api.HandleDns)

		port := 53
		server := &mdns.Server{Addr: ":" + strconv.Itoa(port), Net: "udp"}

		go server.ListenAndServe()
		defer server.Shutdown()

		router := gin.New()

		v1 := router.Group("/api/v1")
		{
			logs := v1.Group("/logs")
			{
				logs.GET(":kind/:group/:identifier", api.Logs)
			}

			operators := v1.Group("/operators")
			{
				operators.GET(":group", api.ListSupported)
				operators.GET(":group/:operator", api.RunOperators)
				operators.POST(":group/:operator", api.RunOperators)
			}

			objects := v1.Group("/")
			{
				objects.POST("apply", api.Apply)
				objects.POST("compare", api.Compare)
				objects.POST("delete", api.Delete)
			}

			containers := v1.Group("/")
			{
				containers.GET("ps", api.Ps)
			}

			database := v1.Group("database")
			{
				database.POST("create/:key", api.DatabaseSet)
				database.PUT("update/:key", api.DatabaseSet)
				database.GET("get/:key", api.DatabaseGet)
				database.GET("keys", api.DatabaseGetKeys)
				database.GET("keys/:prefix", api.DatabaseGetKeysPrefix)
				database.DELETE("keys/:prefix", api.DatabaseRemoveKeys)
			}

			secrets := v1.Group("secrets")
			{
				secrets.GET("get/:secret", api.SecretsGet)
				secrets.GET("keys", api.SecretsGetKeys)
				secrets.POST("create/:secret", api.SecretsSet)
				secrets.PUT("update/:secret", api.SecretsSet)
			}
		}

		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		router.GET("/healthz", api.Health)

		if viper.GetBool("daemon-secured") {
			api.SetupEncryptedDatabase(api.Keys.ServerPrivateKey.Bytes()[:32])

			certPool := x509.NewCertPool()
			if ok := certPool.AppendCertsFromPEM(api.Keys.CAPem.Bytes()); !ok {
				panic("invalid cert in CA PEM")
			}

			serverTLSCert, err := tls.X509KeyPair(api.Keys.ServerCertPem.Bytes(), api.Keys.ServerPrivateKey.Bytes())
			if err != nil {
				logger.Log.Fatal("error opening certificate and key file for control connection", zap.String("error", err.Error()))
				return
			}

			tlsConfig := &tls.Config{
				ClientAuth:   tls.RequireAndVerifyClientCert,
				ClientCAs:    certPool,
				Certificates: []tls.Certificate{serverTLSCert},
			}

			server := http.Server{
				Addr:      ":1443",
				Handler:   router,
				TLSConfig: tlsConfig,
			}

			api.DnsCache.AddARecord(static.SMR_AGENT_DOMAIN, api.Config.Environment.AGENTIP)

			plugins.StartPlugins(api.Config.Root, api.Manager)

			defer server.Close()
			server.ListenAndServeTLS("", "")
		} else {
			router.Run()
		}
	}
}
