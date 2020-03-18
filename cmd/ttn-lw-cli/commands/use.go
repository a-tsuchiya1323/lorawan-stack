// Copyright © 2020 The Things Network Foundation, The Things Industries B.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"crypto/tls"
	"encoding/pem"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.thethings.network/lorawan-stack/pkg/errors"
	"gopkg.in/yaml.v2"
)

func destPath(base string, user bool) (string, error) {
	if !user {
		return base, nil
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, base), nil
}

var (
	errNoHost      = errors.DefineInvalidArgument("no_host", "no host set")
	configFileName = ".ttn-lw-cli.yml"
	caFileName     = "ca.pem"
	useCommand     = &cobra.Command{
		Use:               "use [host]",
		Short:             "Use",
		PersistentPreRunE: preRun(),
		RunE: func(cmd *cobra.Command, args []string) error {
			insecure, _ := cmd.Flags().GetBool("insecure")
			ca, _ := cmd.Flags().GetBool("ca")
			user, _ := cmd.Flags().GetBool("user")
			overwrite, _ := cmd.Flags().GetBool("overwrite")
			credentialsID, _ := cmd.Flags().GetString("credentials-id")

			switch len(args) {
			case 1:
			default:
				return errNoHost
			}

			host := args[0]

			// Build configuration
			conf := MakeDefaultConfig(host, insecure)

			// Credentials ID
			if credentialsID != "" {
				conf.CredentialsID = credentialsID
			}

			// Get CA certificate from server
			if !insecure && ca {
				conn, err := tls.Dial("tcp", conf.NetworkServerGRPCAddress, &tls.Config{InsecureSkipVerify: true})
				if err != nil {
					return err
				}
				defer conn.Close()
				caFile, err := destPath("ca.pem", user)
				if err != nil {
					return err
				}

				f, err := os.Create(caFile)
				defer func() {
					if closeErr := f.Close(); err == nil && closeErr != nil {
						err = closeErr
					}
				}()
				for _, cert := range conn.ConnectionState().PeerCertificates {
					if !cert.IsCA {
						continue
					}
					pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
				}

				conf.CA = caFile
			}

			b, err := yaml.Marshal(conf)
			if err != nil {
				return err
			}

			configFile, err := destPath(configFileName, user)
			if err != nil {
				return err
			}

			_, err = os.Stat(configFile)
			if err == nil && !overwrite {
				logger.Errorf("%s exists. Use --overwrite", configFile)
				os.Exit(-1)
			}

			if err = ioutil.WriteFile(configFile, b, 0644); err != nil {
				logger.Errorf("Failed to write %s: %s\n", configFile, err)
				return err
			}
			logger.Infof("Config file for %s written in %s", host, configFile)
			return nil
		},
	}
)

func init() {
	useCommand.Flags().Bool("insecure", defaultInsecure, "Connect without TLS")
	useCommand.Flags().String("host", defaultClusterHost, "Server host name")
	useCommand.Flags().Bool("ca", false, "Certificate file to use")
	useCommand.Flags().Bool("user", false, "Write config file in user config directory")
	useCommand.Flags().Bool("overwrite", false, "Overwrite existing config files without confirmation")
	Root.AddCommand(useCommand)
}
