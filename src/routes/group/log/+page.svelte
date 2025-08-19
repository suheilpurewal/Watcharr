<script lang="ts">
  import { onMount } from 'svelte';
  let members: Array<{id:string; display_name:string}> = [];
  let selected = new Set<string>();
  let mediaId = '';
  let mediaType = 'movie';
  let startedAt = new Date().toISOString();
  let notes = '';

  onMount(async () => {
    const res = await fetch('/api/members');
    members = await res.json();
  });

  async function save() {
    const attendees = Array.from(selected).map(id => ({ memberId: id, rating: null }));
    const body = { mediaId, mediaType, startedAt, notes: notes || null, attendees };
    const r = await fetch('/api/viewings', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(body) });
    if (r.ok) alert('Saved!');
    else alert('Failed');
  }
</script>

<h1>Log Viewing</h1>

<label>Media ID <input bind:value={mediaId} placeholder="tt1234567 or tvdb:123"/></label>
<label>Type
  <select bind:value={mediaType}>
    <option value="movie">Movie</option>
    <option value="episode">Episode</option>
  </select>
</label>
<label>Date/Time <input type="datetime-local" bind:value={startedAt} /></label>
<label>Notes <input bind:value={notes} /></label>

<h3>Attendees</h3>
{#each members as m}
  <label>
    <input type="checkbox" on:change={(e)=> e.target.checked ? selected.add(m.id) : selected.delete(m.id)} />
    {m.display_name}
  </label>
{/each}

<button on:click={save}>Save</button>
