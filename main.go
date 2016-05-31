// Copyright (C) 2015 Alexander Sokolov <sokoloff.a@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

const (
	version = "0.3.0"
	author  = "Alexander Sokolov <sokoloff.a@gmail.com>"
)

var (
	colorNorm   = "\x1b[0m"
	colorBold   = "\x1b[1m"
	colorGreen  = "\x1b[32m"
	colorYellow = "\x1b[33m"
)

func resetColors() {
	colorNorm = ""
	colorBold = ""
	colorGreen = ""
	colorYellow = ""
}

func colorPrintf(format string, a ...interface{}) {
	format = strings.Replace(format, "{NORM}", colorNorm, -1)
	format = strings.Replace(format, "{BOLD}", colorBold, -1)
	format = strings.Replace(format, "{GREEN}", colorGreen, -1)
	format = strings.Replace(format, "{YELLOW}", colorYellow, -1)

	fmt.Printf(format, a...)
}

func colorSprintf(format string, a ...interface{}) string {
	format = strings.Replace(format, "{NORM}", colorNorm, -1)
	format = strings.Replace(format, "{BOLD}", colorBold, -1)
	format = strings.Replace(format, "{GREEN}", colorGreen, -1)
	format = strings.Replace(format, "{YELLOW}", colorYellow, -1)

	return fmt.Sprintf(format, a...)
}

func machineArch() string {
	utsname := syscall.Utsname{}
	err := syscall.Uname(&utsname)
	if err != nil {
		log.Fatal("Can't get uname:", err)
	}

	s := ""
	for _, v := range utsname.Machine {
		if v > 0 {
			s += string(rune(v))
		}
	}
	return s
}

func getArch(c *cli.Context) []string {
	s := strings.ToLower(c.String("arch"))

	if s == "" {
		return []string{
			"noarch",
			machineArch(),
		}
	}

	if strings.Contains(s, "all") {
		return []string{
			"i586",
			"x86_64",
			"noarch",
		}
	}

	return strings.Split(s, ",")
}

func execute(args ...string) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func checkArgs(c *cli.Context) {
	if len(c.Args()) == 0 {
		fmt.Println("You must provide at least one package name")
		cli.ShowSubcommandHelp(c)
		os.Exit(2)
	}
}

func expandQuery(query []string) []string {
	res := []string{}
	for _, s := range query {
		res = append(res, s)
		if strings.HasPrefix(s, "lib") && !strings.HasPrefix(s, "lib64") {
			res = append(res, "lib64"+s[3:])
		}
	}
	return res
}

func mainRepo(c *cli.Context) {
	showAll := c.Bool("all")

	reps, err := GetRepositories()
	if err != nil {
		log.Fatal("Can't read urpi.cfg file: ", err)
	}

	for _, rep := range reps {
		if rep.Ignore && !showAll {
			continue
		}

		colorPrintf("{BOLD}%s{NORM}\n", rep.Name)
		fmt.Printf("    last update: ")

		if rep.Ignore {
			fmt.Printf("-\n")
		} else {
			fmt.Printf("%v\n", rep.LastUpdate.Format("15:04 01 Jan 2006"))
		}

		fmt.Printf("    URL: %v\n", rep.URL)
		fmt.Println("")
	}
}

func colorizeResultStringOne(q, str string) string {
	if q == "" {
		return str
	}
	q = strings.ToLower(q)

	res := ""
	lStr := strings.ToLower(str)
	for {
		n := strings.Index(lStr, q)
		if n < 0 {
			return res + str
		}

		res += str[:n] + colorBold + str[n:n+len(q)] + colorNorm
		str = str[n+len(q):]
		lStr = lStr[n+len(q):]
	}
}

func colorizeResultString(query []string, str string) string {
	res := str

	for _, q := range query {
		res = colorizeResultStringOne(q, res)
	}
	return res
}

func mainSearch(c *cli.Context) {
	// Query ..........................
	query := c.Args()
	if len(query) == 0 {
		fmt.Println("You must provide at least one search term")
		cli.ShowSubcommandHelp(c)
		return
	}

	// Search .........................
	cache := NewCache()
	query = expandQuery(query)
	out := cache.SearchByName(query, getArch(c), !c.Bool("showduplicates"))

	// Out ............................
	for pkg := range out {
		color := ""
		state := " "
		switch pkg.State() {
		case PACKAGE_INSATALLED:
			color = colorGreen
			state = "I"

		case PACKAGE_UPDATE:
			color = colorYellow
			state = "U"
		}

		line := ""
		line += color + state + colorNorm
		line += colorizeResultString(query, fmt.Sprintf("  %-40s ", pkg.Name))
		line += color + fmt.Sprintf("%-20s", pkg.Version) + colorNorm
		line += fmt.Sprintf("%-8s ", pkg.Arch)
		line += colorizeResultString(query, pkg.Summary)
		fmt.Println(line)
	}
}

func mainShow(c *cli.Context) {
	// Query ..........................
	query := c.Args()
	if len(query) == 0 {
		fmt.Println("You must provide at least one search term")
		cli.ShowSubcommandHelp(c)
		return
	}

	// Search .........................
	cache := NewCache()
	query = expandQuery(query)
	out := cache.SearchByName(query, getArch(c), !c.Bool("showduplicates"))

	// Out ............................
	for pkg := range out {
		colorPrintf("Name        : {BOLD}%s{NORM}\n", pkg.Name)
		colorPrintf("Summary     : %s\n", pkg.Summary)
		colorPrintf("Version     : %-10s %-10s\n", pkg.Version, pkg.Arch)
		switch pkg.State() {
		case PACKAGE_NOTINSATALLED:
			colorPrintf("Not installed\n")

		case PACKAGE_INSATALLED:
			colorPrintf("Installed   : {GREEN}%s{NORM}\n", pkg.InstalledVer)

		case PACKAGE_UPDATE:
			colorPrintf("Installed   : {YELLOW}%s{NORM}\n", pkg.InstalledVer)
		}

		colorPrintf("Group       : %s\n", pkg.Group)
		colorPrintf("Size        : RPM: %v     Files: %v\n", pkg.RPMSize, pkg.Size)
		colorPrintf("Source RPM  : %s\n", pkg.Sourcerpm)
		colorPrintf("URL         : %s\n", pkg.URL)
		colorPrintf("Repository  : %s\n", pkg.Repository)
		s := strings.TrimLeft(pkg.Description, "\n")
		colorPrintf(s)
		fmt.Println("")
	}
}

func prepend(arr []string, a ...string) []string {
	return append(a, arr...)
}

func mainInstall(c *cli.Context) {
	checkArgs(c)
	execute(prepend(c.Args(), "sudo", "urpmi")...)
}

func mainRemove(c *cli.Context) {
	execute(prepend(c.Args(), "sudo", "urpme")...)
}

func mainUpdate(c *cli.Context) {
	execute("sudo", "urpmi.update", "-a")
}

func mainUpgrade(c *cli.Context) {
	execute("sudo", "urpmi", "--auto-select")
}

func mainDownload(c *cli.Context) {
	checkArgs(c)
	execute(prepend(c.Args(), "urpm-downloader", "--binary")...)
}

func mainSource(c *cli.Context) {
	checkArgs(c)
	execute(prepend(c.Args(), "urpm-downloader", "--source")...)
}

func mainFiles(c *cli.Context) {
	checkArgs(c)
	execute(prepend(c.Args(), "urpmf", "-f")...)
}

func mainFile(c *cli.Context) {
	checkArgs(c)
	execute(prepend(c.Args(), "urpmf", "-f")...)
}

func main() {

	app := cli.NewApp()
	app.Name = "zrpm"
	app.Usage = "zrpm is a text-based interface to the RPM package system."
	app.Version = version
	app.Author = author

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "nocolor",
			Usage: "Force black and white output",
		},
	}

	app.Commands = []cli.Command{
		// Repos ......................
		{
			Name:  "repo",
			Usage: "Display information about a repositories.",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "all",
					Usage: "Show disabled repositories too.",
				},
			},
			Action: mainRepo,
		},

		// Search .....................
		{
			Name:      "search",
			Aliases:   []string{"s"},
			Usage:     "Search for a package by name.",
			ArgsUsage: "QUERY...",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name: "arch",
					Usage: "Comma-separated list of architectures (i586, x86_64, noarch). \n\t" +
						"Use 'all' for search packages for any architectures.",
				},

				cli.BoolFlag{
					Name:  "showduplicates",
					Usage: "Doesn't limit packages to their latest versions.",
				},
			},
			Action: mainSearch,
		},

		// Show ............................
		{
			Name:      "show",
			Aliases:   []string{"info"},
			Usage:     "Display detailed information about a package.",
			ArgsUsage: "QUERY...",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name: "arch",
					Usage: "Comma-separated list of architectures (i586, x86_64, noarch). \n\t" +
						"Use 'all' for search packages for any architectures.",
				},

				cli.BoolFlag{
					Name:  "showduplicates",
					Usage: "Doesn't limit packages to their latest versions.",
				},
			},
			Action: mainShow,
		},

		// Install .........................
		{
			Name:      "install",
			Aliases:   []string{"i"},
			Usage:     "Install/upgrade packages.",
			ArgsUsage: "PACKAGE...",
			Action:    mainInstall,
		},

		// Remove .........................
		{
			Name:      "remove",
			Usage:     "Remove packages.",
			ArgsUsage: "PACKAGE...",
			Action:    mainRemove,
		},

		// Update ..........................
		{
			Name:   "update",
			Usage:  "Download lists of new/upgradable packages.",
			Action: mainUpdate,
		},

		// Upgrade .........................
		{
			Name:    "upgrade",
			Aliases: []string{"u"},
			Usage:   "Perform an upgrade, possibly installing and removing packages.",
			Action:  mainUpgrade,
		},

		// Download RPM ....................
		{
			Name:      "download",
			Usage:     "Download binary RPMs.",
			ArgsUsage: "PACKAGE...",
			Action:    mainDownload,
		},

		// Download SRPM ...................
		{
			Name:      "source",
			Usage:     "Download the source RPMs (SRPMs).",
			ArgsUsage: "PACKAGE...",
			Action:    mainSource,
		},

		// List files from SRPM ...........
		{
			Name:      "files",
			Usage:     "List files in package or which package has installed file.",
			ArgsUsage: "PACKAGE...",
			Action:    mainFiles,
		},
	}

	app.CommandNotFound = func(c *cli.Context, command string) {
		fmt.Printf("Unknown command %s\n\n", command)
		cli.ShowSubcommandHelp(c)
	}

	app.Before = func(c *cli.Context) error {
		if c.Bool("nocolor") || !terminal.IsTerminal(int(os.Stdout.Fd())) {
			resetColors()
		}
		return nil
	}

	app.Run(os.Args)
}
