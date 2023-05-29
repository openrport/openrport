package main

import (
	"github.com/realvnc-labs/rport/server/api/authorization"
	"github.com/realvnc-labs/rport/server/clients"
	"github.com/spf13/cobra"
	"log"
)

var (
	migrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "migrate between storage interfaces",
		Long:  "migrate between diffrent storage kinds f.e. from SQLite to FileSystemKeyValue ",
		Run:   migration,
	}
	fromFlag *string
	toFlag   *string
)

func migration(cmd *cobra.Command, args []string) {
	log.Println("started migrating authorization db ")
	err := authorization.Migrate(cfg.Server, *fromFlag, *toFlag)
	if err != nil {
		log.Printf("Migration failed... %v\n", err)
	} else {
		log.Printf("Migration succeded")
	}

	log.Println("started migrating clients db ")
	err = clients.Migrate(cfg.Server, *fromFlag, *toFlag)
	if err != nil {
		log.Printf("Migration failed... %v\n", err)
	} else {
		log.Printf("Migration succeded")
	}

}

func init() {
	RootCmd.AddCommand(migrateCmd)

	fromFlag = migrateCmd.PersistentFlags().StringP("from", "f", "sqlite", "from [required]")
	err := migrateCmd.MarkPersistentFlagRequired("from")
	if err != nil {
		// This will return error if the flag doesn't exist, so it's ok to panic because it can only happen when changing the code
		panic(err)
	}

	toFlag = migrateCmd.PersistentFlags().StringP("to", "t", "fskv", "to [required]")
	err = migrateCmd.MarkPersistentFlagRequired("to")
	if err != nil {
		// This will return error if the flag doesn't exist, so it's ok to panic because it can only happen when changing the code
		panic(err)
	}

	// reset default usage func
	migrateCmd.SetUsageFunc((&cobra.Command{}).UsageFunc())
}
