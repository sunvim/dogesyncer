/*
Copyright © 2022 mobus <sunsc0220@gmail.com>

This program is free software; you can redistribute it and/or
modify it under the terms of the GNU General Public License
as published by the Free Software Foundation; either version 2
of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sunvim/dogesyncer/params"
	"github.com/sunvim/dogesyncer/pkg/server"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "main service entry",
	Long:  ``,
	Run:   server.Run,
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringP(params.ArgGenesis, "g", "", "genesis file,shouldn't null")
	serverCmd.Flags().StringP(params.ArgDatadir, "d", "", "data directory path")

	viper.BindPFlag(params.ArgGenesis, serverCmd.Flags().Lookup(params.ArgGenesis))
	viper.BindPFlag(params.ArgDatadir, serverCmd.Flags().Lookup(params.ArgDatadir))
}
