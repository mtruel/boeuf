# Database Migrations - Strategy

## MVP Approach: GORM AutoMigrate

### Current Implementation

Pour le MVP, le projet utilise **GORM AutoMigrate** pour gérer le schéma de la base de données.

**Activation:** [`backend/cmd/boeuf-server/main.go#L70-L76`](../cmd/boeuf-server/main.go)

```go
// Auto-migrate database schema
if err := db.AutoMigrate(
    &models.Session{},
    &models.SessionInvite{},
    &models.SessionParticipant{},
    &models.SpotifyToken{},
); err != nil {
    log.Fatalf("Failed to migrate database: %v", err)
}
```

### Pourquoi AutoMigrate pour le MVP?

**Avantages:**

- ✅ **Rapidité de développement** : Pas besoin d'écrire des migrations SQL manuellement
- ✅ **Simplicité deploy** : Un seul binaire à déployer, pas de step migration séparé
- ✅ **Synchronisation automatique** : Le schéma DB reflète toujours les structs Go
- ✅ **Prototypage agile** : Changements de modèle rapides sans friction

**Limitations acceptables en MVP:**

- Pas d'historique de migrations versionné
- Migrations DOWN impossibles (rollback manuel requis)
- Contrôle limité sur les détails SQL (indexes, contraintes complexes)

### Migration SQL de Référence

Le fichier [`20260125000001_create_spotify_tokens.sql`](./20260125000001_create_spotify_tokens.sql) existe comme **référence de schéma** et documentation, mais **n'est pas exécuté** par goose.

Ce fichier sert à:

- Documenter le schéma attendu
- Faciliter la transition future vers goose
- Servir de référence pour déploiements manuels si nécessaire

**Note importante pour Story 1.6 (Sync State):** La colonne `sync_state` dans `session_participants` est ajoutée automatiquement par GORM AutoMigrate via le modèle `SessionParticipant`. Aucune migration SQL manuelle n'est requise pour cette feature. Le fichier SQL hypothétique `YYYYMMDDHHMMSS_add_sync_state.sql` mentionné dans les Dev Notes est documentaire uniquement (représente ce que goose exécuterait si activé).

## Future: Migration vers goose

### Quand migrer?

Envisager la migration vers goose (ou autre outil) quand:

- **Production multi-environnements** (dev/staging/prod) nécessite un contrôle versioning strict
- **Équipe grandit** et nécessite un historique de changements tracé
- **Modifications complexes** (renommer colonnes, data migrations) deviennent fréquentes
- **Rollbacks** deviennent critiques pour la production

### Plan de Migration

1. **Capture du schéma actuel** : Exporter le schéma SQLite généré par AutoMigrate
2. **Créer migration initiale goose** : `000001_initial_schema.sql` avec schéma complet
3. **Désactiver AutoMigrate** dans `main.go`
4. **Configurer goose** dans le pipeline de déploiement
5. **Nouvelles migrations** : Créer fichiers goose pour chaque changement futur

### Configuration goose (Référence)

```bash
# Installation
go install github.com/pressly/goose/v3/cmd/goose@latest

# Créer une migration
goose -dir backend/migrations create add_new_column sql

# Appliquer migrations
goose -dir backend/migrations sqlite3 ./data/boeuf.db up

# Rollback
goose -dir backend/migrations sqlite3 ./data/boeuf.db down
```

## Conclusion

**Pour le MVP (Stories 1.x):** GORM AutoMigrate est le choix pragmatique et efficace.

**Pour la production mature:** Transition vers goose sera nécessaire pour contrôle et traçabilité.

Cette stratégie équilibre vélocité de développement MVP et préparation pour scale future.

---

**Last Updated:** 2026-01-29  
**Status:** Active (AutoMigrate)  
**Migration Tool:** GORM v1.31.1
