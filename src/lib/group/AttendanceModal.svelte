<script lang="ts">
  import { onMount, createEventDispatcher } from "svelte";
  const dispatch = createEventDispatcher();

  // Props as signals (read as $open, $mediaId, etc.)
  let {
    open = false,
    mediaId = "",
    mediaType = "movie" as "movie" | "episode",
    defaultStartedAt = "",
    allowRatings = false,
  } = $props();

  type Member = { id: string; displayName: string; isActive: boolean };

  let members: Member[] = [];
  let selected = new Set<string>();
  let startedAt = $defaultStartedAt || new Date().toISOString().slice(0, 16);
  let saving = false;
  let errorMsg = "";

  async function loadMembers() {
    const r = await fetch("/api/group/members");
    members = await r.json();
  }

  // initial fetch if opened on mount
  onMount(() => { if ($open) loadMembers(); });

  // runes effects: READ props with $...
  $effect(() => {
    if ($open) loadMembers();
  });

  $effect(() => {
    if ($open) {
      startedAt = $defaultStartedAt || new Date().toISOString().slice(0, 16);
    }
  });

  function toggle(id: string, checked: boolean) {
    checked ? selected.add(id) : selected.delete(id);
    selected = new Set(selected);
  }

  async function save() {
    console.log("[group] modal save clicked");
    errorMsg = "";
    if (!selected.size) {
      errorMsg = "Pick at least one attendee.";
      return;
    }
    saving = true;
    try {
      const body = {
        mediaId: $mediaId,
        mediaType: $mediaType,
        startedAt: new Date(startedAt).toISOString(),
        notes: null,
        attendees: Array.from(selected).map((id) => ({ memberId: id })),
      };
      const res = await fetch("/api/group/viewings", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) throw new Error(await res.text());
      dispatch("submit");
    } catch (e) {
      errorMsg = (e as Error).message || "Failed to save";
    } finally {
      saving = false;
    }
  }

  function close() {
    console.log("[group] modal cancel clicked");
    dispatch("cancel");
  }
</script>

{#if $open}
  <div class="overlay" onclick={(e) => { if (e.currentTarget === e.target) close(); }}>
    <div class="modal">
      <h3>Who watched?</h3>

      <label class="row">
        <span>When</span>
        <input type="datetime-local" bind:value={startedAt} />
      </label>

      <div class="members">
        {#each members as m}
          {#if m.isActive}
            <label class="chip">
              <input type="checkbox" onchange={(e) => toggle(m.id, (e.target as HTMLInputElement).checked)} />
              <span>{m.displayName}</span>
            </label>
          {/if}
        {/each}
      </div>

      {#if errorMsg}<p class="err">{errorMsg}</p>{/if}

      <div class="actions">
        <button onclick={close} class="ghost">Cancel</button>
        <button onclick={save} disabled={saving}>{saving ? "Saving..." : "Save"}</button>
      </div>
    </div>
  </div>
{/if}



<style>
.overlay{
  position: fixed; inset:0; background: rgba(0,0,0,.4);
  display:flex; align-items:center; justify-content:center; z-index: 1000;
}
.modal{
  width: min(520px, 92vw); background: #111; color:#eee; border-radius: 12px; padding: 16px; box-shadow: 0 10px 30px rgba(0,0,0,.5);
}
.row{ display:flex; align-items:center; gap:10px; margin: 8px 0; }
.row span{ width: 62px; opacity:.8; }
.members{ display:flex; flex-wrap:wrap; gap:8px; margin: 12px 0; }
.chip{ display:flex; align-items:center; gap:6px; background:#1a1a1a; padding:6px 10px; border-radius: 999px; }
.actions{ display:flex; justify-content:flex-end; gap:10px; margin-top: 10px; }
button{ padding:8px 12px; border-radius:8px; border:none; background:#2d6cdf; color:white; cursor:pointer; }
button.ghost{ background:transparent; border:1px solid #444; color:#ddd; }
.err{ color:#ff6b6b; margin: 4px 0 0; }
</style>
