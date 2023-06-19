package cli

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-jsonrpc/auth"

	"github.com/Filecoin-Titan/titan/api/types"
	"github.com/Filecoin-Titan/titan/node/repo"
)

var AuthCmd = &cli.Command{
	Name:  "auth",
	Usage: "Manage RPC permissions",
	Subcommands: []*cli.Command{
		AuthCreateAdminToken,
		AuthAPIInfoToken,
	},
}

var AuthCreateAdminToken = &cli.Command{
	Name:  "create-token",
	Usage: "Create token",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "perm",
			Usage: "permission to assign to the token, one of: web, candidate, edge, locator,admin",
		},
	},

	Action: func(cctx *cli.Context) error {
		napi, closer, err := GetAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)

		if !cctx.IsSet("perm") {
			return xerrors.New("--perm flag not set")
		}

		perm := cctx.String("perm")
		token, err := napi.AuthNew(ctx, &types.JWTPayload{Allow: []auth.Permission{auth.Permission(perm)}, ID: uuid.NewString()})
		if err != nil {
			return err
		}

		// TODO: Log in audit log when it is implemented

		fmt.Println(token)
		return nil
	},
}

var AuthAPIInfoToken = &cli.Command{
	Name:  "api-info",
	Usage: "Get token with API info required to connect to this node",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "id",
			Usage: "special a id for web or candidate, edge, locator,admin",
		},
		&cli.StringFlag{
			Name:  "perm",
			Usage: "permission to assign to the token, one of: web, candidate, edge, locator, admin, web",
		},
	},

	Action: func(cctx *cli.Context) error {
		napi, closer, err := GetAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := ReqContext(cctx)
		id := cctx.String("id")
		perm := cctx.String("perm")
		token, err := napi.AuthNew(ctx, &types.JWTPayload{Allow: []auth.Permission{auth.Permission(perm)}, ID: id})
		if err != nil {
			return err
		}

		ti, ok := cctx.App.Metadata["repoType"]
		if !ok {
			log.Errorf("unknown repo type, are you sure you want to use GetCommonAPI?")
			ti = repo.Scheduler
		}
		t, ok := ti.(repo.RepoType)
		if !ok {
			log.Errorf("repoType type does not match the type of repo.RepoType")
		}

		ainfo, err := GetAPIInfo(cctx, t)
		if err != nil {
			return xerrors.Errorf("could not get API info for %s: %w", t, err)
		}

		// TODO: Log in audit log when it is implemented

		currentEnv, _, _ := t.APIInfoEnvVars()
		fmt.Printf("%s=%s:%s\n", currentEnv, token, ainfo.Addr)
		return nil
	},
}
