package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/troian/semver"
)

type ctxValue string

var (
	ctxValueVersion    = ctxValue("version")
	ctxValueBumpPrerel = ctxValue("prerel")
)

var (
	flagPrefix  = "prefix"
	flagReverse = "reverse"
	flagIsep    = "isep"
	flagOsep    = "osep"
	flagFilter  = "filter"
	flagStdout  = "stdout"
)

var (
	extractRegexp = regexp.MustCompile(`^([.0-9A-Za-z-]*[.A-Za-z-])*([0-9]+)`)
)

func main() {
	viper.AutomaticEnv()
	viper.SetEnvPrefix("SEMVER")

	cmd := &cobra.Command{
		Use:              "semver",
		TraverseChildren: true,
	}

	cmd.AddCommand(
		compareCmd(),
		sortCmd(),
		validateCmd(),
		bumpCmd(),
	)

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func setCmdVersion(cmd *cobra.Command, version *semver.Version) {
	v := cmd.Context().Value(ctxValueVersion)
	if v == nil {
		panic("version context not set")
	}

	valuePtr := v.(*semver.Version)
	*valuePtr = *version
}

func setCmdGenPrerel(cmd *cobra.Command, val bool) {
	v := cmd.Context().Value(ctxValueBumpPrerel)
	if v == nil {
		panic("prerel bump context not set")
	}

	valuePtr := v.(*bool)
	*valuePtr = val
}

func compareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "compare",
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			ver, err := semver.NewVersion(args[0])
			if err != nil {
				return err
			}

			over, err := semver.NewVersion(args[1])
			if err != nil {
				return err
			}

			res := ver.Compare(over)
			fmt.Printf("%d", res)

			return nil
		},
	}

	return cmd
}

func sortCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "sort",
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			reverse := viper.GetBool(flagReverse)
			isep := viper.GetString(flagIsep)
			osep := viper.GetString(flagOsep)
			filter := viper.GetString(flagFilter)

			var input []string

			if len(args) > 0 && args[0] != "-" {
				input = args
			} else {
				res, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return err
				}

				input = strings.Split(strings.TrimSuffix(string(res), "\n"), isep)
			}

			versions := make(semver.Collection, 0, len(input))

			for _, ver := range input {
				res, err := semver.NewVersion(ver)
				if err != nil {
					return err
				}

				versions = append(versions, res)
			}

			sort.Sort(sort.Reverse(versions))

			if filter != "" {
				switch filter {
				case "latest":
					versions = versions[:1]
				case "oldest":
					versions = versions[len(versions)-1:]
				}
			}

			if reverse {
				sort.Sort(sort.Reverse(versions))
			} else {
				sort.Sort(versions)
			}

			var out string
			for i, ver := range versions {
				out += ver.String()

				if i < len(versions)-1 {
					out += osep
				}
			}

			fmt.Printf("%s", out)

			return nil
		},
	}

	cmd.Flags().String(flagIsep, "\n", "input separator")
	cmd.Flags().String(flagOsep, "\n", "output separator")
	cmd.Flags().String(flagFilter, "", "output filter")
	cmd.Flags().BoolP(flagReverse, "r", false, "sort result in reverse order")

	_ = viper.BindPFlags(cmd.Flags())

	return cmd
}

func validateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "validate",
		SilenceUsage: true,
		Args:         cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			isStdout := viper.GetBool(flagStdout)

			var ver string
			if len(args) > 0 && args[0] != "-" {
				ver = args[0]
			} else {
				res, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return err
				}

				ver = strings.TrimSpace(string(res))
			}

			_, err := semver.NewVersion(ver)

			if isStdout {
				if err == nil {
					fmt.Printf("valid")
				} else {
					fmt.Printf("invalid")
				}

				return nil
			}

			return err
		},
	}

	cmd.Flags().Bool(flagStdout, false, "")
	_ = viper.BindPFlags(cmd.Flags())

	return cmd
}

func bumpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bump major|minor|patch|prerel|release",
		Short: "Bump parts of the version",
		Long: `Bump by one of major, minor, patch; zeroing or removing
subsequent parts. "bump prerel" sets the PRERELEASE part
and removes any BUILD part. A trailing dot in the <prerel>
argument introduces an incrementing numeric field
which is added or bumped. If no <prerel> argument is provided, an
incrementing numeric field is introduced/bumped. "bump build" sets
the BUILD part. "bump release" removes any PRERELEASE or BUILD parts.
The bumped version is written to stdout.`,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(2),
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			bVal := new(bool)

			ctx := context.WithValue(cmd.Context(), ctxValueVersion, &semver.Version{})
			ctx = context.WithValue(ctx, ctxValueBumpPrerel, bVal)

			cmd.SetContext(ctx)

			return nil
		},

		PersistentPostRun: func(cmd *cobra.Command, _ []string) {
			ver, valid := cmd.Context().Value(ctxValueVersion).(*semver.Version)
			if !valid {
				return
			}

			bumpPrerel := cmd.Context().Value(ctxValueBumpPrerel).(*bool)
			pchanged := cmd.Flags().Changed(flagPrefix)

			if *bumpPrerel || pchanged {
				res := extractRegexp.FindAllStringSubmatch(ver.Prerelease(), -1)
				var newPrefix string
				var curPrefix string
				var currNum string

				if len(res) > 0 {
					if len(res[0]) > 1 {
						curPrefix = res[0][1]
					}

					if len(res[0]) > 2 {
						currNum = res[0][2]
					}
				}

				if pchanged {
					prefix := viper.GetString(flagPrefix)
					if curPrefix != prefix {
						newPrefix = fmt.Sprintf("%s0", prefix)
					} else if currNum != "" {
						if currNum != "" {
							num, _ := strconv.Atoi(currNum)
							newPrefix = fmt.Sprintf("%s%d", curPrefix, num+1)
						} else {
							newPrefix = fmt.Sprintf("%s0", curPrefix)
						}
					}
				} else {
					if currNum != "" {
						num, _ := strconv.Atoi(currNum)
						newPrefix = fmt.Sprintf("%s%d", curPrefix, num+1)
					} else {
						newPrefix = fmt.Sprintf("%s0", curPrefix)
					}
				}

				ver, _ = ver.SetPrerelease(newPrefix)
			}

			fmt.Printf("%s", ver.String())
		},
	}

	preRunE := func(cmd *cobra.Command, args []string) error {
		version, err := semver.NewVersion(args[0])
		if err != nil {
			return err
		}

		setCmdVersion(cmd, version)

		return nil
	}

	major := &cobra.Command{
		Use:   "major <version>",
		Short: "Bump major part of the version",
		Long: `Bump major; zeroing or removing subsequent parts.
if prefix flag is set, new prerelease will be started.
for example
	command: semver bump major v1.2.3 -p rc
	result: v2.0.0-rc0
`,
		Args:    cobra.ExactArgs(1),
		PreRunE: preRunE,
		Run: func(cmd *cobra.Command, _ []string) {
			ver := cmd.Context().Value(ctxValueVersion).(*semver.Version)

			setCmdVersion(cmd, ver.IncMajor())
		},
	}

	minor := &cobra.Command{
		Use:   "minor <version>",
		Short: "Bump minor part of the version",
		Long: `Bump minor; zeroing or removing subsequent parts.
if prefix flag is set, new prerelease will be started.
for example
	command: semver bump minor v1.2.3 -p rc
	result: v1.3.0-rc0
`,
		Args:    cobra.ExactArgs(1),
		PreRunE: preRunE,
		Run: func(cmd *cobra.Command, _ []string) {
			ver := cmd.Context().Value(ctxValueVersion).(*semver.Version)

			setCmdVersion(cmd, ver.IncMinor())
		},
	}

	patch := &cobra.Command{
		Use:   "patch <version>",
		Short: "Bump patch part of the version",
		Long: `Bump patch; zeroing or removing subsequent parts.
if prefix flag is set, new prerelease will be started.
for example
	command: semver bump patch v1.2.3 -p rc
	result: v1.2.4-rc0
`,
		Args:    cobra.ExactArgs(1),
		PreRunE: preRunE,
		Run: func(cmd *cobra.Command, _ []string) {
			ver := cmd.Context().Value(ctxValueVersion).(*semver.Version)

			setCmdVersion(cmd, ver.IncPatch())
		},
	}

	prerel := &cobra.Command{
		Use:   "prerel <version>",
		Short: "Bump prerel part of the version",
		Long: `Sets the PRERELEASE part and removes any BUILD part.
Prerelease prefix is determined by following rules:
- if there is current prefix set, reuse it and bump number at the end.
  if number is not set, then numeric part starts from 0:             v1.2.3-rc -> v1.2.3-rc0
- if prefix is not detected, tool bumps patch adds numeric part:     v1.2.3    -> v1.2.4-0
Prefix can be specified with prefix flag.
`,
		Args:    cobra.ExactArgs(1),
		PreRunE: preRunE,
		Run: func(cmd *cobra.Command, _ []string) {
			setCmdGenPrerel(cmd, true)
		},
	}

	release := &cobra.Command{
		Use:     "release <version>",
		Short:   "Removes both (if present) PRERELEASE and BUILD parts",
		Args:    cobra.ExactArgs(1),
		PreRunE: preRunE,
		Run: func(cmd *cobra.Command, _ []string) {
			ver := cmd.Context().Value(ctxValueVersion).(*semver.Version)

			ver, _ = ver.SetPrerelease("")
			ver, _ = ver.SetMetadata("")

			setCmdVersion(cmd, ver)
		},
	}

	cmd.AddCommand(
		major,
		minor,
		patch,
		prerel,
		release,
	)

	cmd.PersistentFlags().StringP(flagPrefix, "p", "", "prerelease prefix")
	_ = viper.BindPFlags(cmd.PersistentFlags())

	return cmd
}
