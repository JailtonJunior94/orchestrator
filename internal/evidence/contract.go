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

const ContractCutCommit = "0d84ccd3291c5eb8b762ec7ce6ac766e311dad17"

var contractCut = ContractCutCommit

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

func sealedContentAtCut(reportPath, cut string) ([]byte, bool) {
	if strings.TrimSpace(reportPath) == "" || strings.TrimSpace(cut) == "" {
		return nil, false
	}
	dir := filepath.Dir(reportPath)
	if err := exec.Command("git", "-C", dir, "rev-parse", "--verify", "--quiet", cut+"^{commit}").Run(); err != nil {
		return nil, false
	}
	tracked, err := exec.Command("git", "-C", dir, "ls-files", "--full-name", "--", filepath.Base(reportPath)).Output()
	if err != nil {
		return nil, false
	}
	name := strings.TrimSpace(strings.SplitN(string(tracked), "\n", 2)[0])
	if name == "" {
		return nil, false
	}
	sealed, err := exec.Command("git", "-C", dir, "cat-file", "blob", cut+":"+name).Output()
	if err != nil {
		return nil, false
	}
	return sealed, true
}

func historicalExemption(text, reportPath, cut string) bool {
	sealed, ok := sealedContentAtCut(reportPath, cut)
	if !ok {
		return false
	}
	return string(sealed) == text
}

func historicalExemptionAllowed(text, reportPath string) bool {
	return historicalExemption(text, reportPath, contractCut)
}

func ResolveContract(text, reportPath string) (Contract, []Finding) {
	contract, findings := DetectContract(text)
	if contract != ContractV1 {
		return contract, findings
	}
	if historicalExemptionAllowed(text, reportPath) {
		return ContractV1, findings
	}
	return ContractV2, append(findings, Finding{
		Label: "relatorio sem marcador de contrato nao e evidencia historica — seu conteudo nao corresponde, byte a byte, " +
			"ao blob versionado no commit de corte " + contractCut + ". Trabalho novo (ou relatorio historico editado " +
			"depois do corte) deve declarar '" + ContractMarkerV2 + "' e cumprir as regras estritas (mapa 1:1 de criterios)",
	})
}
