package registration

import (
	"errors"
	"github.com/grussorusso/serverledge/internal/config"
)

var UnavailableClientErr = errors.New("etcd client unavailable")
var IdRegistrationErr = errors.New("etcd error: could not complete the registration")
var KeepAliveErr = errors.New(" The system can't renew your registration key")

type Registry struct {
	Area string
	id   string
}

type ServerInformation struct {
	id   string
	ipv4 string
}

var BASEDIR = "registry"
var TTL = config.GetInt("registry.ttl", 10) // lease time in Seconds
