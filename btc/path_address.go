package btc

import(
	"context"
	"errors"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
	"github.com/hashicorp/vault/helper/salt"
)

type address struct {
	Childnum uint32
	LastAddress string
}

func pathAddress(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "address/" + framework.GenericNameRegex("name"),
		Fields: map[string]*framework.FieldSchema{
			"name": &framework.FieldSchema{
				Type: framework.TypeString,
				Description: "Wallet name",
			},
			"token": &framework.FieldSchema{
				Type: framework.TypeString,
				Description: "Auth token",
			},
		},
		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.UpdateOperation: b.pathAddressWrite,
		},

		HelpSynopsis:    pathAddressHelpSyn,
		HelpDescription: pathAddressHelpDesc,
	}
}

func (b *backend) pathAddressWrite(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	t := d.Get("token").(string)
	if t == "" {
		return nil, errors.New("missing auth token")
	}

	s, err := salt.NewSalt(ctx, req.Storage, nil)
	if err != nil {
		return nil, err
	}

	saltedToken := s.SaltID(t)

	token, err := b.GetToken(ctx, req.Storage, saltedToken)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return logical.ErrorResponse("token not found"), nil
	}

	walletName := token.WalletName

	// get wallet from storage
	w, err := b.GetWallet(ctx, req.Storage, walletName)

	// get last address and address index from storage
	var childnum uint32

	addressEntry, err := req.Storage.Get(ctx, "address/" + walletName)
	if err != nil {
		return nil, err
	}
	// if entry exists calculate next address index
	if addressEntry != nil {
		var a address
		if err := addressEntry.DecodeJSON(&a); err != nil {
			return nil, err
		}
		childnum = a.Childnum + 1
	}

	a, err := deriveAddress(w, childnum)
	if err != nil {
		return nil, err
	}

	// override the storage, is this ok to do in a read operation?
	entry, err := logical.StorageEntryJSON("address/" +  walletName, a)
	if err != nil {
		return nil, err
	}

	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"address": a.LastAddress,
			"childnum": a.Childnum,
		},
	}, nil
}

const pathAddressHelpSyn = `
Returns a new receiving address for selected wallet
`

const pathAddressHelpDesc = `
Test description
`