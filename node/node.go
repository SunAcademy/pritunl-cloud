package node

import (
	"container/list"
	"github.com/Sirupsen/logrus"
	"github.com/dropbox/godropbox/container/set"
	"github.com/pritunl/pritunl-cloud/certificate"
	"github.com/pritunl/pritunl-cloud/constants"
	"github.com/pritunl/pritunl-cloud/database"
	"github.com/pritunl/pritunl-cloud/errortypes"
	"github.com/pritunl/pritunl-cloud/event"
	"github.com/pritunl/pritunl-cloud/settings"
	"github.com/pritunl/pritunl-cloud/utils"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	Self *Node
)

type Node struct {
	Id                 bson.ObjectId              `bson:"_id" json:"id"`
	Zone               bson.ObjectId              `bson:"zone,omitempty" json:"zone"`
	Name               string                     `bson:"name" json:"name"`
	Types              []string                   `bson:"types" json:"types"`
	Timestamp          time.Time                  `bson:"timestamp" json:"timestamp"`
	Port               int                        `bson:"port" json:"port"`
	Protocol           string                     `bson:"protocol" json:"protocol"`
	Hypervisor         string                     `bson:"hypervisor" json:"hypervisor"`
	Certificate        bson.ObjectId              `bson:"certificate" json:"certificate"`
	Certificates       []bson.ObjectId            `bson:"certificates" json:"certificates"`
	AdminDomain        string                     `bson:"admin_domain" json:"admin_domain"`
	UserDomain         string                     `bson:"user_domain" json:"user_domain"`
	RequestsMin        int64                      `bson:"requests_min" json:"requests_min"`
	ForwardedForHeader string                     `bson:"forwarded_for_header" json:"forwarded_for_header"`
	ExternalInterface  string                     `bson:"external_interface" json:"external_interface"`
	InternalInterface  string                     `bson:"internal_interface" json:"internal_interface"`
	Firewall           bool                       `bson:"firewall" json:"firewall"`
	NetworkRoles       []string                   `bson:"network_roles" json:"network_roles"`
	Memory             float64                    `bson:"memory" json:"memory"`
	Load1              float64                    `bson:"load1" json:"load1"`
	Load5              float64                    `bson:"load5" json:"load5"`
	Load15             float64                    `bson:"load15" json:"load15"`
	CpuUnits           int                        `bson:"cpu_units" json:"cpu_units"`
	MemoryUnits        float64                    `bson:"memory_units" json:"memory_units"`
	CpuUnitsRes        int                        `bson:"cpu_units_res" json:"cpu_units_res"`
	MemoryUnitsRes     float64                    `bson:"memory_units_res" json:"memory_units_res"`
	PublicIps          []string                   `bson:"public_ips" json:"public_ips"`
	PublicIps6         []string                   `bson:"public_ips6" json:"public_ips6"`
	SoftwareVersion    string                     `bson:"software_version" json:"software_version"`
	Version            int                        `bson:"version" json:"-"`
	VirtPath           string                     `bson:"virt_path" json:"virt_path"`
	CachePath          string                     `bson:"cache_path" json:"cache_path"`
	CertificateObjs    []*certificate.Certificate `bson:"-" json:"-"`
	reqLock            sync.Mutex                 `bson:"-" json:"-"`
	reqCount           *list.List                 `bson:"-" json:"-"`
}

func (n *Node) AddRequest() {
	n.reqLock.Lock()
	back := n.reqCount.Back()
	back.Value = back.Value.(int) + 1
	n.reqLock.Unlock()
}

func (n *Node) GetVirtPath() string {
	if n.VirtPath == "" {
		return DefaultRoot
	}
	return n.VirtPath
}

func (n *Node) GetCachePath() string {
	if n.CachePath == "" {
		return DefaultCache
	}
	return n.CachePath
}

func (n *Node) IsAdmin() bool {
	for _, typ := range n.Types {
		if typ == Admin {
			return true
		}
	}
	return false
}

func (n *Node) IsUser() bool {
	for _, typ := range n.Types {
		if typ == User {
			return true
		}
	}
	return false
}

func (n *Node) IsHypervisor() bool {
	for _, typ := range n.Types {
		if typ == Hypervisor {
			return true
		}
	}
	return false
}

func (n *Node) Validate(db *database.Database) (
	errData *errortypes.ErrorData, err error) {

	if n.Hypervisor == "" {
		n.Hypervisor = Kvm
	}

	if n.Protocol != "http" && n.Protocol != "https" {
		errData = &errortypes.ErrorData{
			Error:   "node_protocol_invalid",
			Message: "Invalid node server protocol",
		}
		return
	}

	if n.Port < 1 || n.Port > 65535 {
		errData = &errortypes.ErrorData{
			Error:   "node_port_invalid",
			Message: "Invalid node server port",
		}
		return
	}

	if n.Certificates == nil || n.Protocol != "https" {
		n.Certificates = []bson.ObjectId{}
	}

	if n.Types == nil {
		n.Types = []string{}
	}

	if (n.IsAdmin() && !n.IsUser()) || (n.IsUser() && !n.IsAdmin()) {
		n.AdminDomain = ""
		n.UserDomain = ""
	} else {
		if !n.IsAdmin() {
			n.AdminDomain = ""
		}
		if !n.IsUser() {
			n.UserDomain = ""
		}
	}

	if n.Zone != "" {
		coll := db.Zones()
		count, e := coll.FindId(n.Zone).Count()
		if e != nil {
			err = database.ParseError(e)
			return
		}

		if count == 0 {
			n.Zone = ""
		}
	}

	if n.VirtPath == "" {
		n.VirtPath = DefaultRoot
	}
	if n.CachePath == "" {
		n.CachePath = DefaultCache
	}

	if n.NetworkRoles == nil || !n.Firewall {
		n.NetworkRoles = []string{}
	}

	if n.Firewall && len(n.NetworkRoles) == 0 {
		errData = &errortypes.ErrorData{
			Error:   "firewall_empty_roles",
			Message: "Cannot enable firewall without network roles",
		}
		return
	}

	n.Format()

	return
}

func (n *Node) Format() {
	sort.Strings(n.Types)
	utils.SortObjectIds(n.Certificates)
}

func (n *Node) SetActive() {
	if time.Since(n.Timestamp) > 30*time.Second {
		n.RequestsMin = 0
		n.Memory = 0
		n.Load1 = 0
		n.Load5 = 0
		n.Load15 = 0
		n.CpuUnits = 0
		n.CpuUnitsRes = 0
		n.MemoryUnits = 0
		n.MemoryUnitsRes = 0
	}
}

func (n *Node) Commit(db *database.Database) (err error) {
	coll := db.Nodes()

	err = coll.Commit(n.Id, n)
	if err != nil {
		return
	}

	return
}

func (n *Node) CommitFields(db *database.Database, fields set.Set) (
	err error) {

	coll := db.Nodes()

	err = coll.CommitFields(n.Id, n, fields)
	if err != nil {
		return
	}

	return
}

func (n *Node) GetRemoteAddr(r *http.Request) (addr string) {
	if n.ForwardedForHeader != "" {
		addr = strings.TrimSpace(
			strings.SplitN(r.Header.Get(n.ForwardedForHeader), ",", 1)[0])
		if addr != "" {
			return
		}
	}

	addr = utils.StripPort(r.RemoteAddr)
	return
}

func (n *Node) update(db *database.Database) (err error) {
	coll := db.Nodes()

	change := mgo.Change{
		Update: &bson.M{
			"$set": &bson.M{
				"timestamp":        n.Timestamp,
				"requests_min":     n.RequestsMin,
				"memory":           n.Memory,
				"load1":            n.Load1,
				"load5":            n.Load5,
				"load15":           n.Load15,
				"cpu_units":        n.CpuUnits,
				"memory_units":     n.MemoryUnits,
				"cpu_units_res":    n.CpuUnitsRes,
				"memory_units_res": n.MemoryUnitsRes,
				"public_ips":       n.PublicIps,
				"public_ips6":      n.PublicIps6,
			},
		},
		Upsert:    false,
		ReturnNew: true,
	}

	nde := &Node{}

	_, err = coll.Find(&bson.M{
		"_id": n.Id,
	}).Apply(change, nde)
	if err != nil {
		return
	}

	n.Id = nde.Id
	n.Name = nde.Name
	n.Types = nde.Types
	n.Port = nde.Port
	n.Protocol = nde.Protocol
	n.Hypervisor = nde.Hypervisor
	n.Certificates = nde.Certificates
	n.AdminDomain = nde.AdminDomain
	n.UserDomain = nde.UserDomain
	n.ForwardedForHeader = nde.ForwardedForHeader
	n.ExternalInterface = nde.ExternalInterface
	n.InternalInterface = nde.InternalInterface
	n.Firewall = nde.Firewall
	n.NetworkRoles = nde.NetworkRoles
	n.VirtPath = nde.VirtPath
	n.CachePath = nde.CachePath

	return
}

func (n *Node) loadCerts(db *database.Database) (err error) {
	certObjs := []*certificate.Certificate{}

	if n.Certificates == nil || len(n.Certificates) == 0 {
		n.CertificateObjs = certObjs
		return
	}

	for _, certId := range n.Certificates {
		cert, e := certificate.Get(db, certId)
		if e != nil {
			switch e.(type) {
			case *database.NotFoundError:
				e = nil
				break
			default:
				err = e
				return
			}
		} else {
			certObjs = append(certObjs, cert)
		}
	}

	n.CertificateObjs = certObjs

	return
}

func (n *Node) sync() {
	db := database.GetDatabase()
	defer db.Close()

	n.Timestamp = time.Now()

	mem, total, err := utils.MemoryUsed()
	if err != nil {
		n.Memory = 0

		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("node: Failed to get memory")
	} else {
		n.Memory = mem
	}

	load, err := utils.LoadAverage()
	if err != nil {
		n.CpuUnits = 0
		n.MemoryUnits = 0
		n.Load1 = 0
		n.Load5 = 0
		n.Load15 = 0

		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("node: Failed to get load")
	} else {
		n.CpuUnits = load.CpuUnits
		n.MemoryUnits = total
		n.Load1 = load.Load1
		n.Load5 = load.Load5
		n.Load15 = load.Load15
	}

	ipData, err := utils.ExecCombinedOutputLogged(
		[]string{
			"No such file or directory",
			"does not exist",
		},
		"ip", "-f", "inet", "-o", "addr",
		"show", "dev", settings.Local.BridgeName,
	)
	if err != nil {
		return
	}

	fields := strings.Fields(ipData)
	if len(fields) > 3 {
		ipAddr := net.ParseIP(strings.Split(fields[3], "/")[0])
		n.PublicIps = []string{
			ipAddr.String(),
		}
	}

	ipData, err = utils.ExecCombinedOutputLogged(
		[]string{
			"No such file or directory",
			"does not exist",
		},
		"ip", "-f", "inet6", "-o", "addr",
		"show", "dev", settings.Local.BridgeName,
	)
	if err != nil {
		return
	}

	for _, line := range strings.Split(ipData, "\n") {
		if !strings.Contains(line, "global") {
			continue
		}

		fields = strings.Fields(ipData)
		if len(fields) > 3 {
			ipAddr := net.ParseIP(strings.Split(fields[3], "/")[0])
			n.PublicIps6 = []string{
				ipAddr.String(),
			}
		}

		break
	}

	err = n.update(db)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("node: Failed to update node")
	}

	err = n.loadCerts(db)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("node: Failed to load node certificate")
	}
}

func (n *Node) keepalive() {
	for {
		n.sync()
		time.Sleep(1 * time.Second)
	}
}

func (n *Node) reqInit() {
	n.reqLock.Lock()
	n.reqCount = list.New()
	for i := 0; i < 60; i++ {
		n.reqCount.PushBack(0)
	}
	n.reqLock.Unlock()
}

func (n *Node) reqSync() {
	for {
		time.Sleep(1 * time.Second)

		n.reqLock.Lock()

		var count int64
		for elm := n.reqCount.Front(); elm != nil; elm = elm.Next() {
			count += int64(elm.Value.(int))
		}
		n.RequestsMin = count

		n.reqCount.Remove(n.reqCount.Front())
		n.reqCount.PushBack(0)

		n.reqLock.Unlock()
	}
}

func (n *Node) Init() (err error) {
	db := database.GetDatabase()
	defer db.Close()

	coll := db.Nodes()

	err = coll.FindOneId(n.Id, n)
	if err != nil {
		switch err.(type) {
		case *database.NotFoundError:
			err = nil
		default:
			return
		}
	}

	n.SoftwareVersion = constants.Version

	if n.Name == "" {
		n.Name = utils.RandName()
	}

	if n.Types == nil {
		n.Types = []string{Admin, Hypervisor}
	}

	if n.Protocol == "" {
		n.Protocol = "https"
	}

	if n.Port == 0 {
		n.Port = 443
	}

	if n.Hypervisor == "" {
		n.Hypervisor = Kvm
	}

	_, err = coll.UpsertId(n.Id, &bson.M{
		"$set": &bson.M{
			"_id":              n.Id,
			"name":             n.Name,
			"types":            n.Types,
			"timestamp":        time.Now(),
			"protocol":         n.Protocol,
			"port":             n.Port,
			"hypervisor":       n.Hypervisor,
			"software_version": n.SoftwareVersion,
		},
	})
	if err != nil {
		err = database.ParseError(err)
		return
	}

	n.reqInit()

	err = n.loadCerts(db)
	if err != nil {
		return
	}

	event.PublishDispatch(db, "node.change")

	Self = n

	go n.keepalive()
	go n.reqSync()

	return
}
