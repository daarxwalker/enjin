package security

type Policy func(r *Role) bool

func AnyAccount() Policy {
	return func(_ *Role) bool {
		return true
	}
}

func AnyRole() Policy {
	return func(r *Role) bool {
		return r != nil && r.Name != ""
	}
}

func RequireRole(role *Role) Policy {
	return func(r *Role) bool {
		return r != nil && r.Name == role.Name
	}
}

func RequireSecurable(sec Securable) Policy {
	return func(r *Role) bool {
		return r != nil && r.Has(sec)
	}
}

func Require(role *Role, sec Securable) Policy {
	return func(r *Role) bool {
		return r != nil && r.Name == role.Name && r.Has(sec)
	}
}

func Not(policy Policy) Policy {
	return func(r *Role) bool {
		return !policy(r)
	}
}

func Or(policies ...Policy) Policy {
	return func(r *Role) bool {
		if len(policies) == 0 {
			return false
		}
		for _, p := range policies {
			if p(r) {
				return true
			}
		}
		return false
	}
}

func And(policies ...Policy) Policy {
	return func(r *Role) bool {
		if len(policies) == 0 {
			return true
		}
		for _, p := range policies {
			if !p(r) {
				return false
			}
		}
		return true
	}
}
