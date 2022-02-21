package registration

import (
	"errors"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/hexablock/vivaldi"
)

var UnavailableClientErr = errors.New("etcd client unavailable")
var IdRegistrationErr = errors.New("etcd error: could not complete the registration")
var KeepAliveErr = errors.New(" The system can't renew your registration key")

type Registry struct {
	Area   string
	Key    string
	Client *vivaldi.Client
	etcdCh chan bool
}

var BASEDIR = "registry"
var TTL = config.GetInt("registry.ttl", 90) // lease time in Seconds

type StatusInformation struct {
	Url            string
	AvailableMemMB int64
	AvailableCPUs  float64
	DropCount      int64
	Coordinates    vivaldi.Coordinate
}

var serversMap map[string]*StatusInformation
var nearbyServersMap map[string]*StatusInformation
