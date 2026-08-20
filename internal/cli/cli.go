// Package cli provides the RADIUS Director command-line interface.
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/gobcn/radius-director/internal/assets"
	"github.com/gobcn/radius-director/internal/config"
	"github.com/gobcn/radius-director/internal/deployment/docker"
	"github.com/gobcn/radius-director/internal/generator"
	"github.com/gobcn/radius-director/internal/maintenance/accounting"
	"github.com/gobcn/radius-director/internal/output"
	"github.com/gobcn/radius-director/internal/renderer"
	"github.com/gobcn/radius-director/internal/schemas"
	"github.com/gobcn/radius-director/internal/templates"
	"github.com/gobcn/radius-director/internal/validation"
	"github.com/gobcn/radius-director/internal/writer"
)

// Run executes the command-line interface and returns its exit code.
func Run(args []string, stdout, stderr io.Writer, templateLoader templates.Loader, schemaLoader schemas.Loader, runtimeInitializer RuntimeInitializer) int {
	if len(args) > 0 {
		switch args[0] {
		case "init":
			return runInit(args[1:], stdout, stderr, runtimeInitializer)
		case "validate":
			return runValidate(args[1:], stdout, stderr, templateLoader)
		case "generate":
			return runGenerate(args[1:], stdout, stderr, templateLoader, schemaLoader)
		case "maintenance":
			return runMaintenance(args[1:], stdout, stderr, templateLoader)
		case "export":
			return runExport(args[1:], stdout, stderr)
		}
	}

	flags := flag.NewFlagSet("radius-director", flag.ContinueOnError)
	flags.SetOutput(stderr)
	help := flags.Bool("help", false, "Show this help message.")
	flags.BoolVar(help, "h", false, "Show this help message.")
	flags.Usage = func() {
		fmt.Fprintln(stdout, "RADIUS Director manages declarative FreeRADIUS configuration.")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Usage:")
		fmt.Fprintln(stdout, "  radius-director init <runtime-directory> <network-name>")
		fmt.Fprintln(stdout, "  radius-director validate <config.yaml>")
		fmt.Fprintln(stdout, "  radius-director generate <config.yaml> <output-directory>")
		fmt.Fprintln(stdout, "  radius-director maintenance accounting <config.yaml> <tenant>")
		fmt.Fprintln(stdout, "  radius-director export assets [output-directory]")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Options:")
		fmt.Fprintln(stdout, "  -h, --help")
		fmt.Fprintln(stdout, "        Show this help message.")
	}

	if err := flags.Parse(args); err != nil {
		return 2
	}

	if *help {
		flags.Usage()
		return 0
	}

	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected argument: %s\n", flags.Arg(0))
		flags.Usage()
		return 2
	}

	flags.Usage()
	return 0
}

func runInit(
	args []string,
	stdout, stderr io.Writer,
	runtimeInitializer RuntimeInitializer,
) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		printInitUsage(stdout)
		return 0
	}

	if len(args) != 2 {
		fmt.Fprintln(stderr, "init requires a runtime directory and network name")
		printInitUsage(stderr)
		return 2
	}

	if runtimeInitializer == nil {
		fmt.Fprintln(stderr, "runtime initialization is unavailable")
		return 1
	}

	if err := runtimeInitializer.Init(
		context.Background(),
		args[0],
		args[1],
	); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	fmt.Fprintf(
		stdout,
		"RADIUS Director runtime initialized successfully in %s.\n",
		args[0],
	)

	return 0
}

func printInitUsage(output io.Writer) {
	fmt.Fprintln(output, "Usage:")
	fmt.Fprintln(output, "  radius-director init <runtime-directory> <network-name>")
}

func runValidate(args []string, stdout, stderr io.Writer, templateLoader templates.Loader) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(stdout, "Usage:")
		fmt.Fprintln(stdout, "  radius-director validate <config.yaml>")
		return 0
	}

	if len(args) != 1 {
		fmt.Fprintln(stderr, "validate requires exactly one configuration file")
		return 2
	}

	configuration, err := config.Load(args[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if err := validation.Validate(configuration, templateLoader); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	fmt.Fprintln(stdout, "Configuration parsed and validated successfully.")
	return 0
}

func runGenerate(args []string, stdout, stderr io.Writer, templateLoader templates.Loader, schemaLoader schemas.Loader) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(stdout, "Usage:")
		fmt.Fprintln(stdout, "  radius-director generate <config.yaml> <output-directory>")
		return 0
	}

	if len(args) != 2 {
		fmt.Fprintln(stderr, "generate requires a configuration file and output directory")
		return 2
	}

	configuration, err := config.Load(args[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if err := validation.Validate(configuration, templateLoader); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	generated := generator.Generate(configuration)

	templateRenderer := renderer.New(templateLoader)
	generatedOutput, err := output.Build(generated, templateRenderer)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	deployment, err := docker.GenerateDeployment(templateLoader, schemaLoader, generated)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	generatedOutput.Files = append(generatedOutput.Files, deployment.Files...)

	if err := writer.Write(args[1], generatedOutput); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	fmt.Fprintln(stdout, "Configuration generated successfully.")
	return 0
}

func runMaintenance(args []string, stdout, stderr io.Writer, templateLoader templates.Loader) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		printMaintenanceUsage(stdout)
		return 0
	}
	if len(args) == 0 || args[0] != "accounting" {
		fmt.Fprintln(stderr, "maintenance requires the accounting subcommand")
		printMaintenanceUsage(stderr)
		return 2
	}
	return runAccountingMaintenance(args[1:], stdout, stderr, templateLoader)
}

func printMaintenanceUsage(output io.Writer) {
	fmt.Fprintln(output, "Usage:")
	fmt.Fprintln(output, "  radius-director maintenance accounting <config.yaml> <tenant>")
}

func runAccountingMaintenance(args []string, stdout, stderr io.Writer, templateLoader templates.Loader) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		printMaintenanceUsage(stdout)
		return 0
	}
	if len(args) != 2 {
		fmt.Fprintln(stderr, "maintenance accounting requires a configuration file and tenant")
		return 2
	}

	configuration, err := config.Load(args[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := validation.Validate(configuration, templateLoader); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	generated := generator.Generate(configuration)
	tenant, found := generatedTenant(generated, args[1])
	if !found {
		fmt.Fprintf(stderr, "tenant %q does not exist\n", args[1])
		return 1
	}

	enabledPolicies := 0
	for _, policy := range tenant.AccountingPolicies {
		if policy.StaleSessionTimeout != nil {
			enabledPolicies++
		}
	}
	if enabledPolicies == 0 {
		fmt.Fprintf(stdout, "Tenant %q: no stale-session maintenance policies are enabled.\n", tenant.Identifier)
		return 0
	}

	if tenant.SQL.Engine != "mysql" {
		fmt.Fprintf(stderr, "tenant %q: accounting maintenance does not support database engine %q\n", tenant.Identifier, tenant.SQL.Engine)
		return 1
	}

	ctx := context.Background()
	database, err := accounting.OpenMySQL(ctx, tenant.SQL)
	if err != nil {
		fmt.Fprintf(stderr, "tenant %q: %v\n", tenant.Identifier, err)
		return 1
	}
	defer database.Close()

	result, err := (accounting.Runner{DB: database}).Run(ctx, tenant.AccountingPolicies)
	if err != nil {
		fmt.Fprintf(stderr, "tenant %q: accounting maintenance completed with errors: %v\n", tenant.Identifier, err)
		return 1
	}

	fmt.Fprintf(stdout, "Tenant %q: accounting maintenance complete: %d stale session(s) closed across %d enabled NAS policy/policies.\n", tenant.Identifier, result.SessionsClosed, result.PoliciesProcessed)
	return 0
}

func generatedTenant(configuration generator.Configuration, identifier string) (generator.Tenant, bool) {
	for _, tenant := range configuration.Tenants {
		if tenant.Identifier == identifier {
			return tenant, true
		}
	}
	return generator.Tenant{}, false
}

func runExport(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		printExportUsage(stdout)
		return 0
	}

	if len(args) == 0 || args[0] != "assets" {
		fmt.Fprintln(stderr, "export requires the assets subcommand")
		printExportUsage(stderr)
		return 2
	}

	return runExportAssets(args[1:], stdout, stderr)
}

func printExportUsage(output io.Writer) {
	fmt.Fprintln(output, "Usage:")
	fmt.Fprintln(output, "  radius-director export assets <output-directory>")
}

func runExportAssets(args []string, stdout, stderr io.Writer) int {
	return runExportAssetsFromRoot(
		args,
		stdout,
		stderr,
		assets.FactoryRoot,
	)
}

func runExportAssetsFromRoot(args []string, stdout, stderr io.Writer, sourceRoot string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		printExportUsage(stdout)
		return 0
	}

	if len(args) != 1 {
		fmt.Fprintln(stderr, "export assets requires an output directory")
		return 2
	}

	outputDirectory := args[0]

	if err := assets.Export(sourceRoot, outputDirectory); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	fmt.Fprintf(
		stdout,
		"RADIUS Director templates and schemas exported successfully to %s.\n",
		outputDirectory,
	)
	return 0
}

type RuntimeInitializer interface {
	Init(ctx context.Context, root, networkName string) error
}
