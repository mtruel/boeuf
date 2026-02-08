<script setup lang="ts">
import { usePresenceStore } from '@/stores/presence'

const presenceStore = usePresenceStore()
</script>

<template>
  <div class="participants-list">
    <h3 class="participants-title">
      Participants <span class="participant-count">({{ presenceStore.onlineCount }}/{{ presenceStore.participantCount }})</span>
    </h3>
    
    <div v-if="presenceStore.participantCount === 0" class="no-participants">
      No participants yet
    </div>

    <ul v-else class="participants-ul">
      <li
        v-for="participant in presenceStore.participantList"
        :key="participant.userId"
        class="participant-item"
        :class="{ 'participant-online': participant.connectionStatus === 'online', 'participant-offline': participant.connectionStatus === 'offline' }"
      >
        <span class="participant-status-dot" :class="`status-${participant.connectionStatus}`"></span>
        <span class="participant-user-id">{{ participant.userId }}</span>
        <span v-if="participant.role === 'host'" class="participant-role-badge">HOST</span>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.participants-list {
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  padding: 1rem;
  background: #fafafa;
}

.participants-title {
  font-size: 1.125rem;
  font-weight: 600;
  margin: 0 0 0.75rem 0;
  color: #333;
}

.participant-count {
  font-weight: 400;
  color: #666;
  font-size: 0.875rem;
}

.no-participants {
  color: #999;
  font-style: italic;
  padding: 0.5rem 0;
}

.participants-ul {
  list-style: none;
  margin: 0;
  padding: 0;
}

.participant-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem;
  border-radius: 4px;
  margin-bottom: 0.25rem;
}

.participant-item:hover {
  background: #f0f0f0;
}

.participant-status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-online {
  background-color: #4caf50;
}

.status-offline {
  background-color: #999;
}

.participant-user-id {
  flex: 1;
  font-family: monospace;
  font-size: 0.875rem;
  color: #333;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.participant-role-badge {
  background: #2196f3;
  color: white;
  font-size: 0.625rem;
  font-weight: 700;
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.participant-online {
  opacity: 1;
}

.participant-offline {
  opacity: 0.6;
}
</style>
