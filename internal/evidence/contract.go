package evidence

import (
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Contract int

const (
	ContractV1 Contract = 1
	ContractV2 Contract = 2
)

const ContractMarkerV2 = "<!-- evidence-contract: v2 -->"

const contractCutRef = "HEAD"

var contractMarkerRe = regexp.MustCompile(`(?i)<!--\s*evidence-contract\s*:\s*v([0-9]+)\s*-->`)

func DetectContract(text string) (Contract, []Finding) {
	match := contractMarkerRe.FindStringSubmatch(text)
	if match == nil {
		return ContractV1, nil
	}
	version, err := strconv.Atoi(match[1])
	if err != nil || version != int(ContractV2) {
		return ContractV2, []Finding{{Label: "versao de contrato de evidencia desconhecida: v" + match[1] + " (suportado: v2, ou ausencia do marcador para o historico v1)"}}
	}
	return ContractV2, nil
}

func historicalExemptionAllowed(reportPath string) bool {
	if strings.TrimSpace(reportPath) == "" {
		return false
	}
	dir := filepath.Dir(reportPath)
	if err := exec.Command("git", "-C", dir, "rev-parse", "--git-dir").Run(); err != nil {
		return false
	}
	tracked, err := exec.Command("git", "-C", dir, "ls-files", "--full-name", "--", filepath.Base(reportPath)).Output()
	if err != nil {
		return false
	}
	name := strings.TrimSpace(strings.SplitN(string(tracked), "\n", 2)[0])
	if name == "" {
		return false
	}
	return exec.Command("git", "-C", dir, "cat-file", "-e", contractCutRef+":"+name).Run() == nil
}

func ResolveContract(text, reportPath string) (Contract, []Finding) {
	contract, findings := DetectContract(text)
	if contract != ContractV1 {
		return contract, findings
	}
	if historicalExemptionAllowed(reportPath) {
		return ContractV1, findings
	}
	return ContractV2, append(findings, Finding{
		Label: "relatorio sem marcador de contrato nao e evidencia historica — nao esta versionado em " + contractCutRef +
			". Trabalho novo deve declarar '" + ContractMarkerV2 + "' e cumprir as regras estritas (mapa 1:1 de criterios)",
	})
}
