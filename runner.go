package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/jackc/pgx.v2"
	"gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "sqitch-loader"
	app.Usage = "A loader for sqitch database migrations"
	app.Version = "1.0.0"
	app.Flags = []cli.Flag{
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
	}
	app.Action = sqitchAction
	app.Run(os.Args)
}

func sqitchAction(c *cli.Context) error {
	// deploy the schema
	err := deployChado(c)
	if err != nil {
		log.WithFields(log.Fields{
			"type": "deploy-chado",
		}).Error(err)
		return cli.NewExitError(err.Error(), 2)
	}
	log.WithFields(log.Fields{"type": "deploy-chado"}).Info("complete")
	connConfig, err := getConnConfig(c)
	if err != nil {
		return cli.NewExitError(err.Error(), 2)
	}
	conn, err := pgx.Connect(connConfig)
	if err != nil {
		return cli.NewExitError(err.Error(), 2)
	}
	defer conn.Close()
	_, err = conn.Exec("SELECT pg_notify('chado-schema', $1)", "loaded")
	if err != nil {
		return cli.NewExitError(err.Error(), 2)
	}
	log.WithFields(log.Fields{"type": "postgresql notification", "channel": "chado-schema"}).Info("send")
	return nil
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

func getConnConfig(c *cli.Context) (pgx.ConnConfig, error) {
	dsn := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=disable",
		c.GlobalString("chado-user"), c.GlobalString("chado-pass"), c.GlobalString("pghost"),
		c.GlobalString("pgport"), c.GlobalString("chado-db"),
	)
	connConfig, err := pgx.ParseDSN(dsn)
	if err != nil {
		return pgx.ConnConfig{}, err
	}
	return connConfig, nil
}
