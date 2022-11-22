package server

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// OutputFormatter is the standardized interface all output formatters
// should use
type OutputFormatter interface {
	// getErrorOutput returns the CLI command error
	getErrorOutput() string

	// getCommandOutput returns the CLI command output
	getCommandOutput() string

	// SetError sets the encountered error
	SetError(err error)

	// SetCommandResult sets the result of the command execution
	SetCommandResult(result CommandResult)

	// WriteOutput writes the result / error output
	WriteOutput()
}

type CommandResult interface {
	GetOutput() string
}

func shouldOutputJSON(baseCmd *cobra.Command) bool {
	return baseCmd.Flag(JSONOutputFlag).Changed
}

func InitializeOutputter(cmd *cobra.Command) OutputFormatter {
	if shouldOutputJSON(cmd) {
		return newJSONOutput()
	}

	return newCLIOutput()
}

type commonOutputFormatter struct {
	errorOutput   error
	commandOutput CommandResult
}

func (c *commonOutputFormatter) SetError(err error) {
	c.errorOutput = err
}

func (c *commonOutputFormatter) SetCommandResult(result CommandResult) {
	c.commandOutput = result
}

type JSONOutput struct {
	commonOutputFormatter
}

func (jo *JSONOutput) WriteOutput() {
	if jo.errorOutput != nil {
		_, _ = fmt.Fprintln(os.Stderr, jo.getErrorOutput())

		return
	}

	_, _ = fmt.Fprintln(os.Stdout, jo.getCommandOutput())
}

func newJSONOutput() *JSONOutput {
	return &JSONOutput{}
}

func (jo *JSONOutput) getErrorOutput() string {
	return marshalJSONToString(
		struct {
			Err string `json:"error"`
		}{
			Err: jo.errorOutput.Error(),
		},
	)
}

func (jo *JSONOutput) getCommandOutput() string {
	return marshalJSONToString(jo.commandOutput)
}

func marshalJSONToString(input interface{}) string {
	bytes, err := json.Marshal(input)
	if err != nil {
		return err.Error()
	}

	return string(bytes)
}

type CLIOutput struct {
	commonOutputFormatter
}

func newCLIOutput() *CLIOutput {
	return &CLIOutput{}
}

func (cli *CLIOutput) WriteOutput() {
	if cli.errorOutput != nil {
		_, _ = fmt.Fprintln(os.Stderr, cli.getErrorOutput())

		return
	}

	_, _ = fmt.Fprintln(os.Stdout, cli.getCommandOutput())
}

func (cli *CLIOutput) getErrorOutput() string {
	return cli.errorOutput.Error()
}

func (cli *CLIOutput) getCommandOutput() string {
	return cli.commandOutput.GetOutput()
}
