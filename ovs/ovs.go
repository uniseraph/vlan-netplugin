package ovs

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
)

var (
	ovsutils = map[string]string{
		"ovs-vsctl": "",
	}
	supportsXlock = false
	// used to lock iptables commands if xtables lock is not supported
	bestEffortLock sync.Mutex
	// ErrIptablesNotFound is returned when the rule is not found.
	ErrOpenvswitchNotFound     = errors.New("ovs utils  not found")
	ErrOpenvswitchPortNotFound = errors.New("ovs port not found")
)

func initCheck(name string) error {

	if ovsutils[name] == "" {
		path, err := exec.LookPath(name)
		if err != nil {
			return ErrOpenvswitchNotFound
		}
		ovsutils[name] = path
	}
	return nil
}

func Raw(name string, args ...string) (output []byte, err error) {
	if err = initCheck(name); err != nil {
		return nil, err
	}

	for i := 0; i < 30; i++ {
		output, err = exec.Command(ovsutils[name], args...).CombinedOutput()
		logrus.WithFields(
			logrus.Fields{"cmd": "[" + name + " " + strings.Join(args, " ") + "]",
				"output": string(output)},
		).Info("Openvswitch:", err)
		if err == nil {
			break
		}
		if outputstr := string(output); !strings.Contains(outputstr, "database connection failed") &&
			!strings.Contains(outputstr, "is not a bridge or a socket") {
			break
		}
		logrus.Warn("cannot connect to openvswitch database, retry after 1 sec ...")
		time.Sleep(time.Second)
	}

	return output, err
}

func AddBridge(bridge string) error {
	args := []string{"add-br", bridge}
	if _, err := Raw("ovs-vsctl", args...); err != nil {
		return err
	}
	return nil
}

func DelBridge(bridge string) error {
	args := []string{"del-br", bridge}
	if _, err := Raw("ovs-vsctl", args...); err != nil {
		return err
	}
	return nil
}

func ExistsBridge(bridge string) error {
	args := []string{"br-exists", bridge}
	if _, err := Raw("ovs-vsctl", args...); err != nil {
		return err
	}
	return nil
}

func ExistsPort(port string) error {
	args := []string{"port-to-br", port}
	if _, err := Raw("ovs-vsctl", args...); err != nil {
		return err
	}
	return nil
}

func AddVxlanPort(bridge string, port string) (string, error) {
	args := []string{"add-port", bridge, port, "--", "set", "interface", port, "type=vxlan", "options:remote_ip=flow", "options:key=flow"}
	if _, err := Raw("ovs-vsctl", args...); err != nil {
		return "", err
	}

	if ovsPort, err := GetOvsPortNumber(bridge, port); err != nil {
		return "", err
	} else {
		return ovsPort, nil
	}
}

func AddPort(bridge string, port string, options ...string) (string, error) {
	args := []string{"add-port", bridge, port}
	if len(options) > 0 {
		args = append(args, options...)
	}
	if _, err := Raw("ovs-vsctl", args...); err != nil {
		return "", err
	}

	if ovsPort, err := GetOvsPortNumber(bridge, port); err != nil {
		return "", err
	} else {
		return ovsPort, err
	}
}

func DelPort(bridge, port string) error {
	args := []string{"del-port", bridge, port}
	if _, err := Raw("ovs-vsctl", args...); err != nil {
		return err
	}
	return nil
}

func AddFlows(bridge, flowFile string) error {
	args := []string{"add-flows", "-OOpenFlow13", bridge, flowFile}
	if _, err := Raw("ovs-ofctl", args...); err != nil {
		return err
	}
	return nil
}

func DelFlows(bridge, del string) error {
	args := []string{"del-flows", bridge}
	if del != "" {
		args = append(args, del)
	}
	if _, err := Raw("ovs-ofctl", args...); err != nil {
		return err
	}
	return nil
}

func DelFlowsFromFile(bridge, fname string) error {
	c1 := exec.Command("cat", fname)
	c2 := exec.Command("ovs-ofctl", "del-flows", bridge, "-")
	if err := initCheck("ovs-ofctl"); err != nil {
		return err
	}

	r, w := io.Pipe()
	c2.Stdin = r
	c1.Stdout = w
	var b2 bytes.Buffer
	c2.Stdout = &b2
	c2.Stderr = &b2

	if err := c1.Start(); err != nil {
		return err
	}
	if err := c2.Start(); err != nil {
		return err
	}
	if err := c1.Wait(); err != nil {
		return err
	}
	w.Close()
	if err := c2.Wait(); err != nil {
		logrus.WithFields(
			logrus.Fields{"cmd": "[cat" + fname + "|" + "ovs-vsctl del-flows " + bridge + " -" + "]",
				"output": b2.String()},
		).Error("Openvswitch:", err)

		return err
	}

	logrus.WithFields(
		logrus.Fields{"cmd": "[cat " + fname + " | " + " ovs-ofctl del-flows " + bridge + " - " + "]",
			"output": b2.String()},
	).Info("Openvswitch:")

	return nil
}

func DumpFlows(bridge string) (string, error) {
	args := []string{"dump-flows", bridge}
	output, err := Raw("ovs-ofctl", args...)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func ShowBridge(bridge string) error {
	args := []string{"show", bridge}
	_, err := Raw("ovs-ofctl", args...)
	return err
}

/*
func GetOvsPortNumber(bridge string, port string) (string, error) {
	args := []string{"show", bridge}
	output, err := Raw("ovs-ofctl", args...)
	if err != nil {
		return "", err
	}

	for _, ln := range strings.Split(string(output), "\n") {
		if ok := strings.Contains(ln, "("+port+")"); ok {
			index := strings.Index(ln, "("+port+")")
			return strings.TrimSpace(string([]byte(ln)[0:index])), nil
		}
	}
	logger.WithField("portname", port).Error("get ovs port err")
	return "-1", nil
}
*/

func GetOvsPortNumber(_ string, port string) (string, error) {
	args := []string{"get", "Interface", port, "ofport"}
	output, err := Raw("ovs-vsctl", args...)
	if err != nil {
		return "", err
	}
	return strings.Trim(string(output), "\n"), nil
}
