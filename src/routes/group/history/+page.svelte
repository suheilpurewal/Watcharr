<script lang="ts">
  import { onMount } from 'svelte';
  type Row = { sessionID:string; mediaID:string; mediaType:string; startedAt:string; notes?:string; memberName:string; memberID:string; rating?:number };
  let rows: Row[] = [];
  onMount(async () => {
    const res = await fetch('/api/history');
    rows = await res.json();
  });
</script>

<h1>Group History</h1>
<table>
  <thead><tr><th>Date</th><th>Media</th><th>Type</th><th>Member</th><th>Rating</th></tr></thead>
  <tbody>
    {#each rows as r}
      <tr>
        <td>{new Date(r.startedAt).toLocaleString()}</td>
        <td>{r.mediaID}</td>
        <td>{r.mediaType}</td>
        <td>{r.memberName}</td>
        <td>{r.rating ?? '—'}</td>
      </tr>
    {/each}
  </tbody>
</table>
