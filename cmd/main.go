package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/troian/semver"
)

var (
	extractRegexp = regexp.MustCompile(`^([.0-9A-Za-z-]*[.A-Za-z-])*([0-9]+)`)
)

func main() {
	cmd := &cobra.Command{
		Use: "semver",
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
			reverse, err := cmd.Flags().GetBool("reverse")
			if err != nil {
				return err
			}

			isep, err := cmd.Flags().GetString("isep")
			if err != nil {
				return err
			}

			osep, err := cmd.Flags().GetString("osep")
			if err != nil {
				return err
			}

			var input []string

			if len(args) > 0 && args[0] != "-" {
				input = args
			} else {
				res, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return err
				}

				input = strings.Split(string(res), isep)
			}

			versions := make(semver.Collection, 0, len(input))

			for _, ver := range input {
				res, err := semver.NewVersion(ver)
				if err != nil {
					return err
				}

				versions = append(versions, res)
			}

			if reverse {
				sort.Sort(sort.Reverse(versions))
			} else {
				sort.Sort(versions)
			}

			var out string
			for i, ver := range versions {
				out += ver.String()

				if i <= len(versions) {
					out += osep
				}
			}

			fmt.Printf("%s", out)

			return nil
		},
	}

	cmd.Flags().StringP("isep", "s", " ", "")
	cmd.Flags().String("osep", " ", "output separator")
	cmd.Flags().BoolP("reverse", "r", false, "")

	return cmd
}

func validateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "validate",
		SilenceUsage: true,
		Args:         cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			isStdout, err := cmd.Flags().GetBool("stdout")
			if err != nil {
				return err
			}

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

			_, err = semver.NewVersion(ver)

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

	return cmd
}

func bumpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "bump",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			prototype, err := cmd.Flags().GetString("prototype")
			if err != nil {
				return err
			}

			version, err := semver.NewVersion(args[1])
			if err != nil {
				return err
			}

			switch args[0] {
			case "major":
				version.IncMajor()
			case "minor":
				version.IncMinor()
			case "patch":
				version.IncPatch()
			case "prerel":
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

				if prototype == "+" {
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
					if curPrefix != prototype {
						newPrefix = fmt.Sprintf("%s0", prototype)
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

				version = &nVer
			case "build":

			default:

			}

			fmt.Printf("%s", version.String())

			return nil
		},
	}

	cmd.Flags().StringP("prototype", "p", "+", "")

	return cmd
}
