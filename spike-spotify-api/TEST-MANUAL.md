# Test des Endpoints Spotify API

## Instructions pour tester manuellement

### 1. Assure-toi que le serveur tourne

```bash
cd /home/mathias/Documents/boeuf/spike-spotify-api
go run main.go
```

Le serveur devrait afficher :

```
🎵 Spike Spotify API - Server started on http://127.0.0.1:8080
📍 Navigate to http://127.0.0.1:8080/login to authenticate
```

### 2. Authentifie-toi (si pas déjà fait)

Ouvre dans ton navigateur : <http://127.0.0.1:8080/login>

Tu devrais voir un token OAuth après succès.

### 3. Ouvre Spotify sur un appareil

Lance Spotify (mobile, desktop ou web) et joue une musique.

### 4. Teste chaque endpoint

**Dans ton navigateur, visite ces URLs et copie les résultats :**

#### Test 1: Player Status

URL: <http://127.0.0.1:8080/player>

**Ce que tu devrais voir:**

- JSON avec les infos du morceau en cours
- Position de lecture (progress_ms)
- Durée totale
- Nom de l'artiste et du morceau

**Copie le résultat ici :**

```json
[COLLE LA RÉPONSE ICI]
```

---

#### Test 2: Queue

URL: <http://127.0.0.1:8080/queue>

**Ce que tu devrais voir:**

- Liste des morceaux dans la queue
- Morceau actuellement en lecture
- Prochains morceaux

**Copie le résultat ici :**

```json
[COLLE LA RÉPONSE ICI]
```

---

#### Test 3: Pause

URL: <http://127.0.0.1:8080/pause>

**Ce que tu devrais voir:**

- Message: "✅ Pause command sent successfully"
- **Vérifie:** La musique s'est-elle mise en pause sur ton appareil ? OUI / NON

**Résultat :**

```
[COLLE LA RÉPONSE ICI]
```

---

#### Test 4: Play  

URL: <http://127.0.0.1:8080/play>

**Ce que tu devrais voir:**

- Message: "✅ Play command sent successfully"
- **Vérifie:** La musique a-t-elle repris sur ton appareil ? OUI / NON

**Résultat :**

```
[COLLE LA RÉPONSE ICI]
```

---

## 5. Documentation des résultats

Une fois que tu as testé les 4 endpoints, remplace ce fichier avec les résultats copiés, et je génèrerai la documentation finale pour le spike.

## Questions à répondre

1. **Latence perçue** : Combien de temps entre le clic et l'exécution de la commande ? (instantané / 1-2s / 3-5s / >5s)

2. **Fiabilité** : Toutes les commandes ont-elles fonctionné du premier coup ? (OUI / NON)

3. **Informations disponibles** : Les données du player sont-elles suffisantes pour implémenter boeuf ? (OUI / NON)

4. **Limitations observées** : Y a-t-il quelque chose qui ne marche pas comme attendu ?
