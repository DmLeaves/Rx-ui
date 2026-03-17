package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"rxui/internal/model"
)

type desiredRule struct {
	scope    model.FirewallScope
	refID    *int
	port     int
	protocol string
	source   string
	action   string
}

func detectFirewallProvider() string {
	if _, err := exec.LookPath("ufw"); err == nil {
		return "ufw"
	}
	if _, err := exec.LookPath("firewall-cmd"); err == nil {
		return "firewalld"
	}
	if _, err := exec.LookPath("iptables"); err == nil {
		return "iptables"
	}
	return "none"
}

func buildDesiredFirewallRules() []desiredRule {
	rules := make([]desiredRule, 0)

	panelPort, _ := strconv.Atoi(settings["webPort"])
	if panelPort > 0 {
		rules = append(rules, desiredRule{scope: model.FirewallScopePanel, port: panelPort, protocol: "tcp", source: "any", action: "allow"})
	}

	var inbounds []model.Inbound
	db.Where("enable = ?", true).Find(&inbounds)
	for _, inb := range inbounds {
		id := inb.ID
		rules = append(rules, desiredRule{scope: model.FirewallScopeInbound, refID: &id, port: inb.Port, protocol: "tcp", source: "any", action: "allow"})
	}

	var customs []model.FirewallRule
	db.Where("scope = ?", model.FirewallScopeCustom).Find(&customs)
	for _, c := range customs {
		rules = append(rules, desiredRule{scope: c.Scope, refID: c.RefID, port: c.Port, protocol: c.Protocol, source: c.Source, action: c.Action})
	}

	return rules
}

func ruleKey(scope model.FirewallScope, refID *int, port int, protocol string, source string, action string) string {
	ref := "nil"
	if refID != nil {
		ref = strconv.Itoa(*refID)
	}
	return fmt.Sprintf("%s|%s|%d|%s|%s|%s", scope, ref, port, protocol, source, action)
}

func applyRule(provider string, r desiredRule) error {
	portProto := fmt.Sprintf("%d/%s", r.port, r.protocol)
	switch provider {
	case "ufw":
		_, err := exec.Command("ufw", "allow", portProto).CombinedOutput()
		return err
	case "firewalld":
		_, err := exec.Command("firewall-cmd", "--permanent", "--add-port="+portProto).CombinedOutput()
		if err != nil {
			return err
		}
		_, err = exec.Command("firewall-cmd", "--reload").CombinedOutput()
		return err
	case "iptables":
		_, err := exec.Command("iptables", "-C", "INPUT", "-p", r.protocol, "--dport", strconv.Itoa(r.port), "-j", "ACCEPT").CombinedOutput()
		if err == nil {
			return nil
		}
		_, err = exec.Command("iptables", "-I", "INPUT", "-p", r.protocol, "--dport", strconv.Itoa(r.port), "-j", "ACCEPT").CombinedOutput()
		return err
	default:
		return fmt.Errorf("no firewall provider found")
	}
}

func revokeRule(provider string, fr model.FirewallRule) error {
	portProto := fmt.Sprintf("%d/%s", fr.Port, fr.Protocol)
	switch provider {
	case "ufw":
		_, err := exec.Command("ufw", "delete", "allow", portProto).CombinedOutput()
		return err
	case "firewalld":
		_, err := exec.Command("firewall-cmd", "--permanent", "--remove-port="+portProto).CombinedOutput()
		if err != nil {
			return err
		}
		_, err = exec.Command("firewall-cmd", "--reload").CombinedOutput()
		return err
	case "iptables":
		_, err := exec.Command("iptables", "-D", "INPUT", "-p", fr.Protocol, "--dport", strconv.Itoa(fr.Port), "-j", "ACCEPT").CombinedOutput()
		return err
	default:
		return fmt.Errorf("no firewall provider found")
	}
}

func reconcileFirewall() (int, int, error) {
	provider := detectFirewallProvider()
	desired := buildDesiredFirewallRules()

	desiredMap := map[string]desiredRule{}
	for _, d := range desired {
		desiredMap[ruleKey(d.scope, d.refID, d.port, d.protocol, d.source, d.action)] = d
	}

	var existing []model.FirewallRule
	db.Find(&existing)

	existingMap := map[string]model.FirewallRule{}
	for _, e := range existing {
		existingMap[ruleKey(e.Scope, e.RefID, e.Port, e.Protocol, e.Source, e.Action)] = e
	}

	applied, removed := 0, 0
	now := time.Now()

	for k, d := range desiredMap {
		if _, ok := existingMap[k]; ok {
			continue
		}
		fr := model.FirewallRule{Scope: d.scope, RefID: d.refID, Port: d.port, Protocol: d.protocol, Source: d.source, Action: d.action, Provider: provider, Status: model.FirewallStatusPending}
		err := applyRule(provider, d)
		if err != nil {
			fr.Status = model.FirewallStatusFailed
			fr.LastError = err.Error()
		} else {
			fr.Status = model.FirewallStatusApplied
			fr.AppliedAt = &now
			applied++
		}
		db.Create(&fr)
	}

	for k, e := range existingMap {
		if _, ok := desiredMap[k]; ok {
			continue
		}
		err := revokeRule(provider, e)
		if err != nil {
			e.Status = model.FirewallStatusFailed
			e.LastError = strings.TrimSpace(err.Error())
			db.Save(&e)
			continue
		}
		e.Status = model.FirewallStatusStale
		e.LastError = ""
		db.Save(&e)
		removed++
	}

	if provider == "none" {
		return applied, removed, fmt.Errorf("no firewall provider detected")
	}
	return applied, removed, nil
}
