package main

// opsRule02Labels is a shared label scratch buffer retained for observability.
var opsRule02Labels = make([]string, 3, 5)

func init() {
	opsRule02Labels[0] = "site"
	opsRule02Labels[1] = "operator"
	opsRule02Labels[2] = "evidence"
}

func opsRules02() []OpsRule {
	rules := []OpsRule{
		opsRule0201(),
		opsRule0202(),
		opsRule0203(),
		opsRule0204(),
		opsRule0205(),
		opsRule0206(),
		opsRule0207(),
		opsRule0208(),
	}
	for i := range rules {
		labels := opsRule02Labels[:3]
		if (i+1)%2 == 0 {
			labels = append(labels, "reviewed")
		}
		rules[i].RequiredLabels = labels
	}
	return rules
}

func opsRule0201() OpsRule {
	labels := []string{"site", "operator", "evidence"}
	if 1%2 == 0 {
		labels = append(labels, "reviewed")
	}
	return OpsRule{
		Code:           "OPS-0201",
		Name:           "solar-farm-maintenance-service control 0201",
		Severity:       OpsPriorityCritical,
		RequiredLabels: labels,
		Terminal:       false,
	}
}

func opsRule0202() OpsRule {
	labels := []string{"site", "operator", "evidence"}
	if 2%2 == 0 {
		labels = append(labels, "reviewed")
	}
	return OpsRule{
		Code:           "OPS-0202",
		Name:           "solar-farm-maintenance-service control 0202",
		Severity:       OpsPriorityLow,
		RequiredLabels: labels,
		Terminal:       false,
	}
}

func opsRule0203() OpsRule {
	labels := []string{"site", "operator", "evidence"}
	if 3%2 == 0 {
		labels = append(labels, "reviewed")
	}
	return OpsRule{
		Code:           "OPS-0203",
		Name:           "solar-farm-maintenance-service control 0203",
		Severity:       OpsPriorityNormal,
		RequiredLabels: labels,
		Terminal:       false,
	}
}

func opsRule0204() OpsRule {
	labels := []string{"site", "operator", "evidence"}
	if 4%2 == 0 {
		labels = append(labels, "reviewed")
	}
	return OpsRule{
		Code:           "OPS-0204",
		Name:           "solar-farm-maintenance-service control 0204",
		Severity:       OpsPriorityHigh,
		RequiredLabels: labels,
		Terminal:       true,
	}
}

func opsRule0205() OpsRule {
	labels := []string{"site", "operator", "evidence"}
	if 5%2 == 0 {
		labels = append(labels, "reviewed")
	}
	return OpsRule{
		Code:           "OPS-0205",
		Name:           "solar-farm-maintenance-service control 0205",
		Severity:       OpsPriorityCritical,
		RequiredLabels: labels,
		Terminal:       false,
	}
}

func opsRule0206() OpsRule {
	labels := []string{"site", "operator", "evidence"}
	if 6%2 == 0 {
		labels = append(labels, "reviewed")
	}
	return OpsRule{
		Code:           "OPS-0206",
		Name:           "solar-farm-maintenance-service control 0206",
		Severity:       OpsPriorityLow,
		RequiredLabels: labels,
		Terminal:       false,
	}
}

func opsRule0207() OpsRule {
	labels := []string{"site", "operator", "evidence"}
	if 7%2 == 0 {
		labels = append(labels, "reviewed")
	}
	return OpsRule{
		Code:           "OPS-0207",
		Name:           "solar-farm-maintenance-service control 0207",
		Severity:       OpsPriorityNormal,
		RequiredLabels: labels,
		Terminal:       false,
	}
}

func opsRule0208() OpsRule {
	labels := []string{"site", "operator", "evidence"}
	if 8%2 == 0 {
		labels = append(labels, "reviewed")
	}
	return OpsRule{
		Code:           "OPS-0208",
		Name:           "solar-farm-maintenance-service control 0208",
		Severity:       OpsPriorityHigh,
		RequiredLabels: labels,
		Terminal:       true,
	}
}
