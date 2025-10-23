package odata

// OnEntityGet registra um handler para o evento EntityGet
func (s *Server) OnEntityGet(entityName string, handler func(args EventArgs) error) {
	s.eventManager.SubscribeFunc(EventEntityGet, entityName, handler)
}

// OnEntityList registra um handler para o evento EntityList
func (s *Server) OnEntityList(entityName string, handler func(args EventArgs) error) {
	s.eventManager.SubscribeFunc(EventEntityList, entityName, handler)
}

// OnEntityInserting registra um handler para o evento EntityInserting
func (s *Server) OnEntityInserting(entityName string, handler func(args EventArgs) error) {
	s.eventManager.SubscribeFunc(EventEntityInserting, entityName, handler)
}

// OnEntityInserted registra um handler para o evento EntityInserted
func (s *Server) OnEntityInserted(entityName string, handler func(args EventArgs) error) {
	s.eventManager.SubscribeFunc(EventEntityInserted, entityName, handler)
}

// OnEntityModifying registra um handler para o evento EntityModifying
func (s *Server) OnEntityModifying(entityName string, handler func(args EventArgs) error) {
	s.eventManager.SubscribeFunc(EventEntityModifying, entityName, handler)
}

// OnEntityModified registra um handler para o evento EntityModified
func (s *Server) OnEntityModified(entityName string, handler func(args EventArgs) error) {
	s.eventManager.SubscribeFunc(EventEntityModified, entityName, handler)
}

// OnEntityDeleting registra um handler para o evento EntityDeleting
func (s *Server) OnEntityDeleting(entityName string, handler func(args EventArgs) error) {
	s.eventManager.SubscribeFunc(EventEntityDeleting, entityName, handler)
}

// OnEntityDeleted registra um handler para o evento EntityDeleted
func (s *Server) OnEntityDeleted(entityName string, handler func(args EventArgs) error) {
	s.eventManager.SubscribeFunc(EventEntityDeleted, entityName, handler)
}

// OnEntityError registra um handler para o evento EntityError
func (s *Server) OnEntityError(entityName string, handler func(args EventArgs) error) {
	s.eventManager.SubscribeFunc(EventEntityError, entityName, handler)
}

// OnEntityGetGlobal registra um handler global para o evento EntityGet
func (s *Server) OnEntityGetGlobal(handler func(args EventArgs) error) {
	s.eventManager.SubscribeGlobalFunc(EventEntityGet, handler)
}

// OnEntityListGlobal registra um handler global para o evento EntityList
func (s *Server) OnEntityListGlobal(handler func(args EventArgs) error) {
	s.eventManager.SubscribeGlobalFunc(EventEntityList, handler)
}

// OnEntityInsertingGlobal registra um handler global para o evento EntityInserting
func (s *Server) OnEntityInsertingGlobal(handler func(args EventArgs) error) {
	s.eventManager.SubscribeGlobalFunc(EventEntityInserting, handler)
}

// OnEntityInsertedGlobal registra um handler global para o evento EntityInserted
func (s *Server) OnEntityInsertedGlobal(handler func(args EventArgs) error) {
	s.eventManager.SubscribeGlobalFunc(EventEntityInserted, handler)
}

// OnEntityModifyingGlobal registra um handler global para o evento EntityModifying
func (s *Server) OnEntityModifyingGlobal(handler func(args EventArgs) error) {
	s.eventManager.SubscribeGlobalFunc(EventEntityModifying, handler)
}

// OnEntityModifiedGlobal registra um handler global para o evento EntityModified
func (s *Server) OnEntityModifiedGlobal(handler func(args EventArgs) error) {
	s.eventManager.SubscribeGlobalFunc(EventEntityModified, handler)
}

// OnEntityDeletingGlobal registra um handler global para o evento EntityDeleting
func (s *Server) OnEntityDeletingGlobal(handler func(args EventArgs) error) {
	s.eventManager.SubscribeGlobalFunc(EventEntityDeleting, handler)
}

// OnEntityDeletedGlobal registra um handler global para o evento EntityDeleted
func (s *Server) OnEntityDeletedGlobal(handler func(args EventArgs) error) {
	s.eventManager.SubscribeGlobalFunc(EventEntityDeleted, handler)
}

// OnEntityErrorGlobal registra um handler global para o evento EntityError
func (s *Server) OnEntityErrorGlobal(handler func(args EventArgs) error) {
	s.eventManager.SubscribeGlobalFunc(EventEntityError, handler)
}
