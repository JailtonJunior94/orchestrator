package detect

func (d *ToolchainDetector) detectJava(projectDir string) (ToolchainEntry, bool) {
	if len(d.findManifests(projectDir, "pom.xml")) > 0 {
		return ToolchainEntry{
			Test: "mvn test",
			Lint: "mvn verify",
		}, true
	}
	if len(d.findManifests(projectDir, "build.gradle")) > 0 || len(d.findManifests(projectDir, "build.gradle.kts")) > 0 {
		return ToolchainEntry{
			Test: "./gradlew test",
			Lint: "./gradlew check",
		}, true
	}
	return ToolchainEntry{}, false
}
