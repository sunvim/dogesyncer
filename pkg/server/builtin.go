package server

import (
	"github.com/sunvim/dogesyncer/secrets"
	"github.com/sunvim/dogesyncer/secrets/awsssm"
	"github.com/sunvim/dogesyncer/secrets/hashicorpvault"
	"github.com/sunvim/dogesyncer/secrets/local"
)

// secretsManagerBackends defines the SecretManager factories for different
// secret management solutions
var secretsManagerBackends = map[secrets.SecretsManagerType]secrets.SecretsManagerFactory{
	secrets.Local:          local.SecretsManagerFactory,
	secrets.HashicorpVault: hashicorpvault.SecretsManagerFactory,
	secrets.AWSSSM:         awsssm.SecretsManagerFactory,
}
