package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/team-croffle/croffle/apps/server/internal/config"
	"github.com/team-croffle/croffle/apps/server/internal/secretbox"
)

// maxKeyInput bounds what --new-key-stdin reads: a key is 44 characters.
const maxKeyInput = 4096

func (c *command) secret() error {
	switch action := c.action(); action {
	case "generate":
		return c.secretGenerate()
	case "rotate":
		return c.secretRotate()
	default:
		return c.usage(
			"croffle-server secret generate",
			"croffle-server secret rotate --new-key-file <path> | --new-key-stdin [--old-key-file <path>]")
	}
}

// secretGenerate prints a new key. It does not store it anywhere: where the
// key lives (a Docker secret, a file next to the compose file) is the
// operator's decision, and it must not be the database volume.
func (c *command) secretGenerate() error {
	// --db is accepted like everywhere else, but no database is opened.
	positional, err := parseInterspersed(c.flags("generate", false), c.args)
	if err != nil {
		return err
	}
	if len(positional) > 0 {
		return c.usage("croffle-server secret generate")
	}
	key, err := secretbox.GenerateKey()
	if err != nil {
		return err
	}
	c.printf("%s\n", key)
	return nil
}

// rotateWarning is printed on every rotate: the command cannot tell whether a
// server is running on the same database, and a running server keeps sealing
// new enrolments with the old key until it restarts.
const rotateWarning = `Stop the server before rotating. A running server keeps using the old key until
it restarts, so a one-time code enrolled in between is sealed with the old key and
the next start fails. To recover from that, configure the new key and run
  croffle-server secret rotate --new-key-file <new key> --old-key-file <old key>
which opens secrets sealed with either key and seals them all with the new one.
`

// secretRotate re-seals every TOTP secret with a new key, in one transaction.
// The current key is the configured one (CROFFLE_SECRET_KEY or
// CROFFLE_SECRET_KEY_FILE); without one, the plain secrets are sealed for the
// first time.
func (c *command) secretRotate() error {
	fs := c.flags("rotate", false)
	newKeyFile := fs.String("new-key-file", "", "file holding the new key")
	newKeyStdin := fs.Bool("new-key-stdin", false, "read the new key from stdin")
	oldKeyFile := fs.String("old-key-file", "", "another key the stored secrets may be sealed with (recovery)")
	if err := c.parse(fs); err != nil {
		return err
	}
	if len(c.args) > 0 {
		return c.usage("croffle-server secret rotate --new-key-file <path> | --new-key-stdin")
	}

	var raw []byte
	var err error
	switch {
	case *newKeyFile != "" && *newKeyStdin:
		return errors.New("give --new-key-file or --new-key-stdin, not both")
	case *newKeyFile != "":
		if raw, err = os.ReadFile(*newKeyFile); err != nil {
			return fmt.Errorf("read the new key: %w", err)
		}
	case *newKeyStdin:
		if raw, err = io.ReadAll(io.LimitReader(c.in, maxKeyInput)); err != nil {
			return fmt.Errorf("read the new key: %w", err)
		}
	default:
		return errors.New("give the new key with --new-key-file <path> or --new-key-stdin (make one with `croffle-server secret generate`)")
	}
	newKey, err := secretbox.ParseKey(string(raw))
	if err != nil {
		return fmt.Errorf("the new key: %w", err)
	}
	to, err := secretbox.New(newKey)
	if err != nil {
		return err
	}

	currentKey, err := config.LoadSecretKey(c.getenv)
	if err != nil {
		return fmt.Errorf("the current key: %w", err)
	}
	var sources []*secretbox.AEAD
	if currentKey != nil {
		box, err := secretbox.New(currentKey)
		if err != nil {
			return err
		}
		sources = append(sources, box)
	}
	if *oldKeyFile != "" {
		raw, err := os.ReadFile(*oldKeyFile)
		if err != nil {
			return fmt.Errorf("read the old key: %w", err)
		}
		oldKey, err := secretbox.ParseKey(string(raw))
		if err != nil {
			return fmt.Errorf("the old key: %w", err)
		}
		box, err := secretbox.New(oldKey)
		if err != nil {
			return err
		}
		sources = append(sources, box)
	}
	// Re-sealing with the key everything already uses is pointless, unless an
	// old key is given to recover secrets sealed with it.
	if *oldKeyFile == "" && len(sources) == 1 && sources[0].KeyID() == to.KeyID() {
		return fmt.Errorf("the new key is the current key (id %s)", to.KeyID())
	}
	var from secretbox.Box = secretbox.Plain
	if len(sources) > 0 {
		from = secretbox.Keys(sources...)
	}

	fmt.Fprint(c.errOut, rotateWarning)
	count, err := c.db.RotateSecrets(c.ctx, from, to)
	if err != nil {
		return fmt.Errorf("nothing was changed: %w", err)
	}
	c.printf("sealed %d one-time code secret(s) with key %s\n", count, to.KeyID())
	c.printf("Now set %s (or %s) to the new key and start the server.\n",
		config.EnvSecretKey, config.EnvSecretKeyFile)
	return nil
}
