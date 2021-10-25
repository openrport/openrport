package main

import (
	chserver "github.com/cloudradar-monitoring/rport/server"
	"github.com/spf13/cobra"
	"os"
)

var totpCmd = &cobra.Command{
	Use:   "totp",
	Short: "Generates a time based one time password with an optional qr code image",
	RunE: func(cmd *cobra.Command, args []string) error {
		imagePath, err := cmd.Flags().GetString("totp-image")
		if err != nil {
			return err
		}

		accountName, err := cmd.Flags().GetString("totp-account")
		if err != nil {
			return err
		}

		issuer, err := cmd.Flags().GetString("totp-issuer")
		if err != nil {
			return err
		}

		return chserver.GenerateTotPSecretKey(issuer, accountName, imagePath, os.Stdout)
	},
}

func initTotP(rootCmd *cobra.Command) {
	totpCmd.Flags().StringP(
		"totp-image",
		"",
		"",
		"path for a qr code image which can be used in an authenticator app, you can use jpg or png extension to specify image format",
	)
	totpCmd.Flags().StringP(
		"totp-account",
		"",
		"rport",
		"Account name, you can use your email here",
	)
	totpCmd.Flags().StringP(
		"totp-issuer",
		"s",
		"rport",
		"Issuer of the secret key",
	)

	rootCmd.AddCommand(totpCmd)
}
