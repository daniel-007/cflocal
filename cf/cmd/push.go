package cmd

import (
	"errors"
	"flag"
	"fmt"
)

type Push struct {
	UI     UI
	App    App
	FS     FS
	Help   Help
	Config Config
}

type pushOptions struct {
	name               string
	keepState, pushEnv bool
}

func (p *Push) Match(args []string) bool {
	return len(args) > 0 && args[0] == "push"
}

func (p *Push) Run(args []string) error {
	options, err := p.options(args)
	if err != nil {
		if err := p.Help.Show(); err != nil {
			p.UI.Error(err)
		}
		return err
	}

	if err := p.pushDroplet(options.name); err != nil {
		return err
	}
	if options.pushEnv {
		if err := p.pushEnv(options.name); err != nil {
			return err
		}
	}
	if !options.keepState {
		if err := p.App.Restart(options.name); err != nil {
			return err
		}
	}
	p.UI.Output("Successfully pushed: %s", options.name)
	return nil
}

func (p *Push) pushDroplet(name string) error {
	droplet, size, err := p.FS.ReadFile(fmt.Sprintf("./%s.droplet", name))
	if err != nil {
		return err
	}
	defer droplet.Close()
	return p.App.SetDroplet(name, droplet, size)
}

func (p *Push) pushEnv(name string) error {
	localYML, err := p.Config.Load()
	if err != nil {
		return err
	}
	return p.App.SetEnv(name, getAppConfig(name, localYML).Env)
}

func (*Push) options(args []string) (*pushOptions, error) {
	set := &flag.FlagSet{}
	options := &pushOptions{name: args[1]}
	set.BoolVar(&options.keepState, "k", false, "")
	set.BoolVar(&options.pushEnv, "e", false, "")
	if err := set.Parse(args[2:]); err != nil {
		return nil, err
	}
	if set.NArg() != 0 {
		return nil, errors.New("invalid arguments")
	}
	return options, nil
}
