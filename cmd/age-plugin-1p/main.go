package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"filippo.io/age"
	page "filippo.io/age/plugin"
	"github.com/dirkhusemann/age-plugin-1p/plugin"
	"github.com/spf13/cobra"
)

type PluginOptions struct {
	AgePlugin       string
	Convert         bool
	Generate        string
	OutputFile      string
	LogFile         string
	PrintRecipients bool
}

var example = `
  $ age-plugin-1p --print-recipients
  op://Personal/SSH key/public key: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINKZfejb9htpSB5K9p0RuEowErkba2BMKaze93ZVkQIE

  $ echo "Hello World" | age -r "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINKZfejb9htpSB5K9p0RuEowErkba2BMKaze93ZVkQIE" > secret.age

  $ age --decrypt -j 1p -o - secret.age
  Hello World`

var (
	pluginOptions = PluginOptions{}
	rootCmd       = &cobra.Command{
		Use:     "age-plugin-1p",
		Long:    "age-plugin-1p is a tool to generate age compatible identities backed by SSH keys stored in 1Password.",
		Example: example,
		RunE:    RunPlugin,
	}
)

func SetLogger() {
	var w io.Writer
	if pluginOptions.LogFile != "" {
		w, _ = os.Open(pluginOptions.LogFile)
	} else if os.Getenv("AGEDEBUG") != "" {
		w = os.Stderr
	} else {
		w = io.Discard
	}
	plugin.SetLogger(w)
}

func RunCli(cmd *cobra.Command, in io.Reader, out io.Writer) error {
	switch {
	case pluginOptions.PrintRecipients:
		output, err := plugin.MarshalAllRecipients()
		if err != nil {
			return err
		}
		fmt.Fprint(out, output)
	case pluginOptions.Generate != "":

		if pluginOptions.OutputFile != "" && pluginOptions.OutputFile != "-" {
			f, err := os.OpenFile(pluginOptions.OutputFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if err != nil {
				return err
			}
			defer f.Close()
			out = f
		}

		identity, err := plugin.CreateIdentityFromPath(pluginOptions.Generate)
		if err != nil {
			return err
		}
		err = identity.Recipient().MarshalWithDefaultIdentity(out)
		if err != nil {
			return err
		}
	case pluginOptions.Convert:
		identity, err := plugin.ParseIdentity(in)
		if err != nil {
			return err
		}
		recipient := identity.Recipient()
		return plugin.MarshalRecipient(recipient, out)
	default:
		return cmd.Help()
	}
	return nil
}

func RunPlugin(cmd *cobra.Command, args []string) error {

	switch pluginOptions.AgePlugin {
	case "recipient-v1":
		plugin.Log.Println("Got recipient-v1")
		p, err := page.New(plugin.PluginName)
		if err != nil {
			return err
		}
		p.HandleRecipient(func(data []byte) (age.Recipient, error) {
			r, err := plugin.DecodeRecipient(page.EncodeRecipient(plugin.PluginName, data))
			if err != nil {
				return nil, err
			}
			return r, nil
		})
		p.HandleIdentityAsRecipient(func(data []byte) (age.Recipient, error) {
			i, err := plugin.DecodeIdentity(page.EncodeIdentity(plugin.PluginName, data))
			if err != nil {
				return nil, err
			}
			return i.Recipient(), nil
		})
		if exitCode := p.RecipientV1(); exitCode != 0 {
			return fmt.Errorf("age-plugin exited with code %d", exitCode)
		}
	case "identity-v1":
		plugin.Log.Println("Got identity-v1")
		p, err := page.New(plugin.PluginName)
		if err != nil {
			return err
		}
		p.HandleIdentity(func(data []byte) (age.Identity, error) {
			// someone passed the default identity using `age --decrypt -j op`
			if data == nil {
				return plugin.NewDefaultIdentity()
			}

			i, err := plugin.DecodeIdentity(page.EncodeIdentity(plugin.PluginName, data))
			if err != nil {
				return nil, err
			}
			return i, nil
		})
		if exitCode := p.IdentityV1(); exitCode != 0 {
			return fmt.Errorf("age-plugin exited with code %d", exitCode)
		}
	default:
		in := os.Stdin
		if inFile := cmd.Flags().Arg(0); inFile != "" && inFile != "-" {
			f, err := os.Open(inFile)
			if err != nil {
				return fmt.Errorf("failed to open input file %q: %v", inFile, err)
			}
			defer f.Close()
			in = f
		}
		return RunCli(cmd, in, os.Stdout)
	}
	return nil
}

func pluginFlags(cmd *cobra.Command, opts *PluginOptions) {
	flags := cmd.Flags()
	flags.SortFlags = false

	flags.BoolVar(&opts.PrintRecipients, "print-recipients", false, "Print all the public keys in 1Password")

	flags.BoolVarP(&opts.Convert, "convert", "y", false, "Print recipient for identity file passed through stdin")
	flags.StringVarP(&opts.OutputFile, "output", "o", "", "Write the result to the file at path `OUTPUT`")

	flags.StringVarP(&opts.Generate, "generate", "g", "", "Generate an identity file for SSH key at 1Password CLI `REFERENCE` e.g. \"op://vault/item/private key\"")

	flags.StringVar(&opts.LogFile, "log-file", "", "Write logs to `FILE`")

	flags.StringVar(&opts.AgePlugin, "age-plugin", "", "internal use")
	flags.MarkHidden("age-plugin")
}

func main() {
	SetLogger()
	pluginFlags(rootCmd, &pluginOptions)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
