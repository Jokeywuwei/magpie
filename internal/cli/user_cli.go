// magpie user: the platform's accounts, made and managed from a terminal.
// The first admin is made here: there is no default account or password,
// and none is ever written into a file magpie ships.
package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/yetone/magpie/internal/identity"
)

const userUsage = `  magpie user list [--json]                the platform's accounts, without their hashes
  magpie user add <email> [--name <name>] [--role user|admin] [--password-stdin]
                                           make an account; the password is asked for, never typed as an argument.
                                           The first account must be an admin.
  magpie user passwd <email> [--password-stdin]   change a password and sign its other browsers out
  magpie user role <email> user|admin      an account's role
  magpie user disable <email>              stop it signing in, and end its sessions
  magpie user enable <email>               let it sign in again
  magpie user remove <email> --yes         delete it and its sessions
  magpie user registration [on|off]        whether visitors may sign up in the user portal;
                                           off until an administrator turns it on

  A password is 8 to 1024 characters. Typed on a terminal it is not echoed, and
  add asks for it twice; piped in, one line is read (--password-stdin says so).
  Users sign in to the web page with this account; the API keys they call models
  with are made in the console, and are not this password.
`

func userCmd(args []string) error {
	if len(args) == 0 {
		args = []string{"list"}
	}
	switch args[0] {
	case "help", "-h", "--help":
		fmt.Print(userUsage)
		return nil
	case "list", "ls":
		return userList(args[1:])
	case "add", "new":
		return userAdd(args[1:])
	case "passwd", "password":
		return userPasswd(args[1:])
	case "role":
		return userRole(args[1:])
	case "disable", "off":
		return userSetDisabled(args[1:], true)
	case "enable", "on":
		return userSetDisabled(args[1:], false)
	case "remove", "rm", "delete", "del":
		return userRemove(args[1:])
	case "registration", "register":
		return userRegistration(args[1:])
	}
	return fmt.Errorf("unknown user command %q\n\n%s", args[0], userUsage)
}

func userList(args []string) error {
	asJSON := false
	for _, a := range args {
		switch a {
		case "--json", "-j":
			asJSON = true
		default:
			return fmt.Errorf("usage: magpie user list [--json]")
		}
	}
	users, err := identity.List()
	if err != nil {
		return err
	}
	if asJSON {
		b, err := json.MarshalIndent(users, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}
	if len(users) == 0 {
		fmt.Println(muted.Render("  no accounts yet · magpie user add <email> --role admin"))
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "EMAIL\tROLE\tNAME\tSTATUS\tCREATED\tLAST SIGN-IN")
	for _, u := range users {
		status := "on"
		if u.Disabled {
			status = "disabled"
		}
		last := "never"
		if u.LastLogin != nil {
			last = u.LastLogin.Local().Format("2006-01-02 15:04")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", u.Email, u.Role, u.Name, status, u.CreatedAt.Local().Format("2006-01-02"), last)
	}
	return w.Flush()
}

// userArgs is the arguments of add and passwd: the email and the flags.
func userArgs(args []string) (email, name, role string, stdin bool, rest []string, err error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--name" && i+1 < len(args):
			i++
			name = args[i]
		case strings.HasPrefix(a, "--name="):
			name = strings.TrimPrefix(a, "--name=")
		case a == "--role" && i+1 < len(args):
			i++
			role = args[i]
		case strings.HasPrefix(a, "--role="):
			role = strings.TrimPrefix(a, "--role=")
		case a == "--password-stdin":
			stdin = true
		case strings.HasPrefix(a, "-"):
			return "", "", "", false, nil, fmt.Errorf("unknown flag %s", a)
		case email == "":
			email = a
		default:
			rest = append(rest, a)
		}
	}
	return email, name, role, stdin, rest, nil
}

func userAdd(args []string) error {
	email, name, role, fromStdin, rest, err := userArgs(args)
	if err != nil {
		return err
	}
	if email == "" || len(rest) > 0 {
		return fmt.Errorf("usage: magpie user add <email> [--name <name>] [--role user|admin] [--password-stdin]")
	}
	r, err := identity.ParseRole(role)
	if err != nil {
		return err
	}
	password, err := accountPassword(fromStdin, true)
	if err != nil {
		return err
	}
	u, err := identity.Add(email, name, password, r)
	if err != nil {
		return err
	}
	fmt.Printf("✓ %s %s (%s)\n", "added", u.Email, u.Role)
	return nil
}

func userPasswd(args []string) error {
	email, _, _, fromStdin, rest, err := userArgs(args)
	if err != nil {
		return err
	}
	if email == "" || len(rest) > 0 {
		return fmt.Errorf("usage: magpie user passwd <email> [--password-stdin]")
	}
	u, err := identity.ByEmail(email)
	if err != nil {
		return err
	}
	password, err := accountPassword(fromStdin, true)
	if err != nil {
		return err
	}
	if err := identity.SetPassword(u.ID, password); err != nil {
		return err
	}
	fmt.Printf("✓ %s %s\n", "changed the password of", u.Email)
	return nil
}

func userRole(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: magpie user role <email> user|admin")
	}
	u, err := identity.ByEmail(args[0])
	if err != nil {
		return err
	}
	r, err := identity.ParseRole(args[1])
	if err != nil {
		return err
	}
	if _, err := identity.SetRole(u.ID, r); err != nil {
		return err
	}
	fmt.Printf("✓ %s %s\n", u.Email, r)
	return nil
}

func userSetDisabled(args []string, disabled bool) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: magpie user %s <email>", map[bool]string{true: "disable", false: "enable"}[disabled])
	}
	u, err := identity.ByEmail(args[0])
	if err != nil {
		return err
	}
	if _, err := identity.SetDisabled(u.ID, disabled); err != nil {
		return err
	}
	said := map[bool]string{true: "disabled", false: "enabled"}[disabled]
	fmt.Printf("✓ %s %s\n", u.Email, said)
	return nil
}

func userRemove(args []string) error {
	email, yes := "", false
	for _, a := range args {
		switch {
		case a == "--yes" || a == "-y":
			yes = true
		case strings.HasPrefix(a, "-"):
			return fmt.Errorf("unknown flag %s", a)
		case email == "":
			email = a
		default:
			return fmt.Errorf("usage: magpie user remove <email> --yes")
		}
	}
	if email == "" {
		return fmt.Errorf("usage: magpie user remove <email> --yes")
	}
	if !yes {
		return fmt.Errorf("removing %s deletes the account and signs its browsers out: add --yes", email)
	}
	u, err := identity.ByEmail(email)
	if err != nil {
		return err
	}
	if err := identity.Remove(u.ID); err != nil {
		return err
	}
	fmt.Printf("✓ %s %s\n", "removed", u.Email)
	return nil
}

// userRegistration shows or sets whether visitors may make accounts in the
// user portal. The first administrator is always made with user add, never
// by registration.
func userRegistration(args []string) error {
	if len(args) == 0 {
		open, err := identity.RegistrationOpen()
		if err != nil {
			return err
		}
		said := "closed"
		if open {
			said = "open"
		}
		fmt.Printf("registration is %s\n", said)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: magpie user registration [on|off]")
	}
	var open bool
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "on", "open", "true", "yes":
		open = true
	case "off", "closed", "false", "no":
		open = false
	default:
		return fmt.Errorf("usage: magpie user registration [on|off]")
	}
	if err := identity.SetRegistrationOpen(open); err != nil {
		return err
	}
	said := map[bool]string{true: "open", false: "closed"}[open]
	fmt.Printf("✓ registration is %s\n", said)
	return nil
}

// accountPassword reads a password: unseen and twice on a terminal, one
// line from a pipe or with --password-stdin. A password is never an
// argument, where a shell's history and the process list would keep it.
func accountPassword(fromStdin, confirm bool) (string, error) {
	if fromStdin {
		line, err := stdin().ReadString('\n')
		return strings.TrimRight(line, "\r\n"), err
	}
	if promptTerminal(int(os.Stdin.Fd())) {
		fmt.Fprint(os.Stderr, "Password: ")
		b, err := promptHidden(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		if confirm {
			fmt.Fprint(os.Stderr, "Password again: ")
			again, err := promptHidden(int(os.Stdin.Fd()))
			fmt.Fprintln(os.Stderr)
			if err != nil {
				return "", err
			}
			if !bytes.Equal(b, again) {
				return "", errors.New("the passwords differ")
			}
		}
		return string(b), nil
	}
	line, err := stdin().ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
