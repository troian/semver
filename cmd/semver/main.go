package main

import (
	"context"
	"errors"
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
	ctxValueVersion = ctxValue("version")
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

func setCmdVersion(cmd *cobra.Command, version *semver.Version) error {
	v := cmd.Context().Value(ctxValueVersion)
	if v == nil {
		return errors.New("client context not set")
	}

	versionPtr := v.(*semver.Version)
	*versionPtr = *version

	return nil
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
			reverse := viper.GetBool("reverse")
			isep := viper.GetString("isep")
			osep := viper.GetString("osep")
			filter := viper.GetString("filter")

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

	cmd.Flags().String("isep", "\n", "input separator")
	cmd.Flags().String("osep", "\n", "output separator")
	cmd.Flags().String("filter", "", "output filter")
	cmd.Flags().BoolP("reverse", "r", false, "")

	_ = viper.BindPFlags(cmd.Flags())

	return cmd
}

func validateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "validate",
		SilenceUsage: true,
		Args:         cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			isStdout := viper.GetBool("stdout")

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

	cmd.Flags().Bool("stdout", false, "")
	_ = viper.BindPFlags(cmd.Flags())

	return cmd
}

func bumpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "bump",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(2),
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			ctx := context.WithValue(cmd.Context(), ctxValueVersion, &semver.Version{})
			cmd.SetContext(ctx)

			return nil
		},

		PersistentPostRun: func(cmd *cobra.Command, _ []string) {
			ver, valid := cmd.Context().Value(ctxValueVersion).(*semver.Version)
			if !valid {
				return
			}

			fmt.Printf("%s", ver.String())
		},
	}

	preRunE := func(cmd *cobra.Command, args []string) error {
		version, err := semver.NewVersion(args[0])
		if err != nil {
			return err
		}

		return setCmdVersion(cmd, version)
	}

	major := &cobra.Command{
		Use:     "major",
		Args:    cobra.ExactArgs(1),
		PreRunE: preRunE,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ver := cmd.Context().Value(ctxValueVersion).(*semver.Version)

			return setCmdVersion(cmd, ver.IncMajor())
		},
	}

	minor := &cobra.Command{
		Use:     "minor",
		Args:    cobra.ExactArgs(1),
		PreRunE: preRunE,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ver := cmd.Context().Value(ctxValueVersion).(*semver.Version)

			return setCmdVersion(cmd, ver.IncMinor())
		},
	}

	patch := &cobra.Command{
		Use:     "patch",
		Args:    cobra.ExactArgs(1),
		PreRunE: preRunE,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ver := cmd.Context().Value(ctxValueVersion).(*semver.Version)

			return setCmdVersion(cmd, ver.IncPatch())
		},
	}

	prerel := &cobra.Command{
		Use:     "prerel",
		Args:    cobra.ExactArgs(1),
		PreRunE: preRunE,
		RunE: func(cmd *cobra.Command, _ []string) error {
			prefix := viper.GetString("prefix")
			version := cmd.Context().Value(ctxValueVersion).(*semver.Version)

			res := extractRegexp.FindAllStringSubmatch(version.Prerelease(), -1)
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

			if prefix == "+" {
				if currNum != "" {
					num, err := strconv.Atoi(currNum)
					if err != nil {
						return err
					}
					newPrefix = fmt.Sprintf("%s%d", curPrefix, num+1)
				} else {
					newPrefix = fmt.Sprintf("%s0", curPrefix)
				}
			} else {
				if curPrefix != prefix {
					newPrefix = fmt.Sprintf("%s0", prefix)
				} else if currNum != "" {
					if currNum != "" {
						num, err := strconv.Atoi(currNum)
						if err != nil {
							return err
						}
						newPrefix = fmt.Sprintf("%s%d", curPrefix, num+1)
					} else {
						newPrefix = fmt.Sprintf("%s0", curPrefix)
					}
				}
			}

			nVer, err := version.SetPrerelease(newPrefix)
			if err != nil {
				return err
			}

			return setCmdVersion(cmd, nVer)
		},
	}

	cmd.AddCommand(
		major,
		minor,
		patch,
		prerel,
	)

	prerel.Flags().StringP("prefix", "p", "+", "")
	_ = viper.BindPFlags(prerel.Flags())

	return cmd
}
