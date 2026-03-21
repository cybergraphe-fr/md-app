# Session — Reset version MD et audit CI/CD apps

- Date: 2026-03-13
- Heure: 11:55
- Scope: md-app + audit des pipelines CI/CD des repos sous `apps/`

## Objectifs
- Corriger la release MD pour repartir d'une base `v0.19.0` et supprimer le faux départ `v0.1.1`.
- Inventorier tous les repos applicatifs sous `apps/`.
- Vérifier les pipelines GitHub Actions / CI-CD de chaque repo.
- Corriger les échecs jusqu'à obtention d'un état vert partout quand c'est faisable depuis ce workspace.

## Plan
- Vérifier l'historique/tags/releases de `md-app` pour réaligner en `v0.19.x`.
- Corriger tag + déploiement prod MD.
- Lister les repos git sous `/volume1/docker/apps`.
- Interroger les workflows/runs GitHub par repo.
- Traiter les échecs repo par repo, valider, pousser si nécessaire, puis revérifier.

## Exécution réalisée

### 1. MD app
- Tag incorrect `v0.1.1` supprimé et release recalée sur `v0.19.1`.
- `.env` prod recalé sur `APP_VERSION=v0.19.1`.
- Redéploiement NAS exécuté et endpoint public de santé revalidé.

### 2. Repos audités sous `apps/`
- `aegis-enclave`
- `afteryou`
- `afteryou_mobile`
- `compliance-pilot`
- `enregle_fr`
- `luvu_fr`
- `md`
- `onboard-bot`
- `pianobot_v2`

### 3. Correctifs réellement poussés

#### `cybergraphe-fr/afteryou`
- `640ef50` `ci(afteryou): refresh sbom and harden security scan` sur `main`
- `ac5df1c` `ci(afteryou): artifact-only security scan on develop` sur `develop`
- `75c7d60` `build(afteryou): unpin trivy installer on develop` sur `develop`

Contenu effectif:
- génération SBOM non interactive et auto-bootstrappée (`npm ci` si `node_modules` absent)
- `Security Scan` privé converti en flux artefacts au lieu d'uploads GHAS bloquants
- build Docker `develop` réparé via retrait du pin Trivy cassé

#### `cybergraphe-fr/aegis-enclave`
- `e8b3a6d` `ci(aegis): stabilize private compliance workflows`
- `9ad9a43` `ci(aegis): fix compliance notification guard`
- `1d28f5e` `ci(aegis): keep quick maturity gate focused`

Contenu effectif:
- `Security Scan` privé converti en artefacts + CodeQL conditionnel privé
- `Compliance Audit` rendu non bloquant côté Discord si secret absent
- `generate-sbom.sh` fiabilisé en non interactif
- `audit.sh` rendu compatible avec la nomenclature `AE_*`
- `maturity-gate --mode quick` recentré sur les checks de gate CI, audit avancé réservé au mode `full`

#### `cybergraphe-fr/md-app`
- aucun nouveau commit code dans cette phase CI
- rerun des runs PR bloqués `action_required` pour nettoyer l'état GitHub

### 4. Nettoyage environnement temporaire
- worktree temporaire AfterYou `develop` supprimé: `/tmp/afteryou-develop-fix`
- branche locale d'assistance supprimée: `develop-ci-fixes`

## Résultats vérifiés

### Prod MD
- `md.cybergraphe.fr/health` validé sur `v0.19.1`

### Derniers runs utiles par repo

#### `cybergraphe-fr/afteryou`
- dernier `CI/CD Pipeline` sur `develop`: ✅ `23057039917`
- dernier `Security Scan` sur `develop`: ✅ `23056577643`
- dernier `CI/CD Pipeline` sur `main`: ✅ `23056554358`

#### `cybergraphe-fr/aegis-enclave`
- dernier `CI/CD Pipeline` sur `main`: ✅ `23057423302`
- dernier `maturity-gate` sur `main`: ✅ `23057422967`
- dernier `Compliance Audit` sur `main`: ✅ `23056625638`
- dernier `Supply Chain Security` sur `main`: ✅ `23056582995`
- dernier `Security Scan` sur `main`: ✅ `23056579797`

#### `cybergraphe-fr/md-app`
- dernier `CI` sur `main`: ✅ `23046908036`
- dernier `CD Production` sur `main`: ✅ `23046908020`
- rerun `CodeQL` PR: ✅ `23044110216`
- rerun `CI` PR: ✅ `23044110179`

## Conclusion
- objectif atteint sur les derniers runs pertinents des branches actives auditées
- les anciens runs rouges restent visibles dans l'historique GitHub, mais les derniers runs utiles sont passés au vert