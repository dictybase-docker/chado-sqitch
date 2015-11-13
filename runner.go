package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
	"gopkg.in/codegangsta/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "sqitch-loader"
	app.Usage = "A loader for sqitch database migrations"
	app.Version = "1.0.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "etcd-host",
			EnvVar: "ETCD_CLIENT_SERVICE_HOST",
			Usage:  "ip address of etcd instance",
		},
		cli.StringFlag{
			Name:   "etcd-port",
			EnvVar: "ETCD_CLIENT_SERVICE_PORT",
			Usage:  "port number of etcd instance",
		},
		cli.StringFlag{
			Name:   "chado-pass",
			EnvVar: "CHADO_PASS",
			Usage:  "chado database password",
		},
		cli.StringFlag{
			Name:   "chado-db",
			EnvVar: "CHADO_DB",
			Usage:  "chado database name",
		},
		cli.StringFlag{
			Name:   "chado-user",
			EnvVar: "CHADO_USER",
			Usage:  "chado database user",
		},
		cli.StringFlag{
			Name:   "pghost",
			EnvVar: "POSTGRES_SERVICE_HOST",
			Usage:  "postgresql host",
		},
		cli.StringFlag{
			Name:   "pgport",
			EnvVar: "POSTGRES_SERVICE_PORT",
			Usage:  "postgresql port",
		},
		cli.StringFlag{
			Name:  "key-register",
			Usage: "key to register after finish of loading",
			Value: "/migration/sqitch",
		},
		cli.StringFlag{
			Name:  "key-watch",
			Usage: "key to watch before start loading",
			Value: "/migration/postgresql",
		},
	}
	app.Action = sqitchAction
	app.Run(os.Args)
}

func sqitchAction(c *cli.Context) {
	// wait for etcd key
	if len(c.String("etcd-host")) > 1 && len(c.String("etcd-port")) > 1 {
		client, err := getEtcdAPIHandler(c)
		if err != nil {
			log.WithFields(log.Fields{
				"type": "etcd-client",
			}).Fatal(err)
		}
		err = waitForEtcd(client, c)
		if err != nil {
			log.WithFields(log.Fields{
				"type": "etcd-client",
			}).Fatal(err)
		}
		log.WithFields(log.Fields{
			"type": "etcd-client",
			"key":  c.String("key-watch"),
		}).Info("finished watching key")
	}
	// extract database credentials, only in case of kubernetes
	err := extractSecret()
	if err != nil {
		log.WithFields(log.Fields{
			"type": "secret",
		}).Fatal(err)
	}
	log.WithFields(log.Fields{
		"type": "secrets",
	}).Info("finised going through secrets")

	// deploy the schema
	err = deployChado(c)
	if err != nil {
		log.WithFields(log.Fields{
			"type": "deploy-chado",
		}).Fatal(err)
	}
	log.WithFields(log.Fields{"type": "deploy-chado"}).Info("complete")

	// register the completion with etcd
	if len(c.String("etcd-host")) > 1 && len(c.String("etcd-port")) > 1 {
		client, err := getEtcdAPIHandler(c)
		if err != nil {
			log.WithFields(log.Fields{
				"type": "etcd-client",
			}).Fatal(err)
		}
		err = registerWithEtcd(client, c)
		if err != nil {
			log.WithFields(log.Fields{
				"type": "etcd-client",
			}).Fatal(err)
		}
		log.WithFields(log.Fields{
			"type": "etcd-client",
			"key":  c.String("register-key"),
		}).Info("register key")
	}
}

func getEtcdURL(c *cli.Context) string {
	return "http://" + c.String("etcd-host") + ":" + c.String("etcd-port")
}

func getEtcdAPIHandler(c *cli.Context) (client.KeysAPI, error) {
	cfg := client.Config{
		Endpoints: []string{getEtcdURL(c)},
		Transport: client.DefaultTransport,
	}
	cl, err := client.New(cfg)
	if err != nil {
		return nil, err
	}
	return client.NewKeysAPI(cl), nil
}

func waitForEtcd(api client.KeysAPI, c *cli.Context) error {
	_, err := api.Get(context.Background(), c.String("key-watch"), nil)
	if err != nil {
		// key is not present have to watch it
		if m, _ := regexp.MatchString("100", err.Error()); m {
			w := api.Watcher(c.String("key-watch"), nil)
			_, err := w.Next(context.Background())
			if err != nil {
				return err
			}
			return nil
		} else {
			return err
		}
	}
	// key is already present
	return nil
}

func extractSecret() error {
	sc := map[string]string{
		"/secrets/chadouser": "CHADO_USER",
		"/secrets/chadopass": "CHADO_PASS",
		"/secrets/chadodb":   "CHADO_DB",
	}
	for k, v := range sc {
		if b, err := readSecretFile(k); err != nil {
			os.Setenv(v, string(b))
		} else {
			return err
		}
	}
	return nil
}

func readSecretFile(path string) ([]byte, error) {
	if _, err := os.Stat(path); os.IsExist(err) {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return b, nil
	}
	return nil, nil
}

func deployChado(c *cli.Context) error {
	sqitch, err := exec.LookPath("sqitch")
	if err != nil {
		return err
	}
	psql, err := exec.LookPath("psql")
	if err != nil {
		return err
	}
	config := []string{
		"config",
		"--user",
		"engine.pg.client",
		psql,
	}
	// add the psql client path to sqitch
	cb, err := exec.Command(sqitch, config...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s\t%s\t%s", err.Error(), string(cb), strings.Join(config, " "))
	}

	if len(c.String("chado-user")) > 1 && len(c.String("chado-db")) > 1 && len(c.String("chado-pass")) > 1 {
		if len(c.String("pghost")) > 1 {
			// check if postgres connection is alive before
			// deploying
			if err := waitForPostgres(c); err != nil {
				return err
			}

			dburi := fmt.Sprintf("db:pg://%s:%s@%s:%s/%s",
				c.String("chado-user"),
				c.String("chado-pass"),
				c.String("pghost"),
				c.String("pgport"),
				c.String("chado-db"),
			)
			// add an target uri with a name
			target := []string{"target", "add", "dictychado", dburi}
			tb, err := exec.Command(sqitch, target...).CombinedOutput()
			if err != nil {
				return fmt.Errorf("%s\t%s\t%s", err.Error(), string(tb), strings.Join(target, " "))
			}

			// deploy to the target uri
			deploy := []string{"deploy", "-t", "dictychado"}
			dpb, err := exec.Command(sqitch, deploy...).CombinedOutput()
			if err != nil {
				return fmt.Errorf("%s\t%s\t%s", err.Error(), string(dpb), strings.Join(deploy, " "))
			}
			log.WithFields(log.Fields{
				"type":  "sqitch-client",
				"stage": "deploy",
			}).Info(string(dpb))

		} else {
			return fmt.Errorf("no postgres host is defined")
		}
		return nil
	}
	return fmt.Errorf("does not have any information about database to deploy")
}

func registerWithEtcd(api client.KeysAPI, c *cli.Context) error {
	_, err := api.Create(context.Background(), c.String("key-register"), "complete")
	if err != nil {
		return err
	}
	return nil
}

func waitForPostgres(c *cli.Context) error {
	log.WithFields(log.Fields{
		"type": "postgres-client",
	}).Info("Going to check for database connection")
	time.Sleep(10000 * time.Millisecond)
	for {
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", c.String("pghost"), c.String("pgport")))
		if err == nil {
			log.WithFields(log.Fields{
				"type": "postgres-client",
			}).Info("Postgresql database started")
			conn.Close()
			time.Sleep(10000 * time.Millisecond)
			return nil
		}
		log.WithFields(log.Fields{
			"type": "postgres-client",
		}).Warn("Postgresql database not started, going to recheck ....")
		time.Sleep(10000 * time.Millisecond)
		conn.Close()
	}
	return nil
}
