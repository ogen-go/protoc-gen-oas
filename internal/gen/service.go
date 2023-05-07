package gen

import "google.golang.org/protobuf/compiler/protogen"

// NewService return Service instance.
func NewService(s *protogen.Service) (*Service, error) {
	n := string(s.Desc.Name())

	methods, err := NewMethods(s.Methods)
	if err != nil {
		return nil, err
	}

	return &Service{
		Name:    n,
		Methods: methods,
	}, nil
}

// NewServices returns Services instance.
func NewServices(ss []*protogen.Service) (Services, error) {
	services := make(Services, 0, len(ss))

	for _, s := range ss {
		service, err := NewService(s)
		if err != nil {
			return nil, err
		}

		services = append(services, service)
	}

	return services, nil
}

// Service instance.
type Service struct {
	Name    string
	Methods Methods
}

// Services is Service slice instance.
type Services []*Service
