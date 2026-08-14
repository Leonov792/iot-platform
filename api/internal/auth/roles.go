package auth

// Роли доступа. owner — полный доступ к контуру, family — климат/свет,
// staff — только бассейн/спортзал в заданные часы.
const (
	RoleOwner  = "owner"
	RoleFamily = "family"
	RoleStaff  = "staff"
)

// ValidRole проверяет, что строка — одна из известных ролей.
func ValidRole(role string) bool {
	switch role {
	case RoleOwner, RoleFamily, RoleStaff:
		return true
	default:
		return false
	}
}
