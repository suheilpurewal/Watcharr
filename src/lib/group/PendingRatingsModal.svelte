<script lang="ts">
  import { onMount } from "svelte";
  import { store } from "@/store.svelte";

  export let open = false;
  export let onClose = () => {};

  type PendingRating = {
    attendanceId: string;
    sessionId: string;
    mediaId: string;
    mediaType: string;
    startedAt: string;
    notes?: string;
  };

  let pendingRatings: PendingRating[] = [];
  let loading = false;
  let currentRatingIndex = 0;
  let currentRating = 5;
  let submitting = false;

  async function loadPendingRatings() {
    if (!open) return;
    
    loading = true;
    try {
      const response = await fetch("/api/group/my-pending-ratings", {
        headers: {
          "Authorization": store.userInfo?.token || "",
        },
      });
      
      if (response.ok) {
        pendingRatings = await response.json();
        currentRatingIndex = 0;
      } else {
        console.error("Failed to load pending ratings");
      }
    } catch (error) {
      console.error("Error loading pending ratings:", error);
    } finally {
      loading = false;
    }
  }

  async function submitRating() {
    if (currentRatingIndex >= pendingRatings.length) return;
    
    const rating = pendingRatings[currentRatingIndex];
    submitting = true;
    
    try {
      const response = await fetch(`/api/group/attendance/${rating.attendanceId}/rating`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          "Authorization": store.userInfo?.token || "",
        },
        body: JSON.stringify({ rating: currentRating }),
      });
      
      if (response.ok) {
        // Move to next rating or close modal
        currentRatingIndex++;
        currentRating = 5; // Reset to default
        
        if (currentRatingIndex >= pendingRatings.length) {
          // All ratings submitted, close modal
          onClose();
        }
      } else {
        console.error("Failed to submit rating");
      }
    } catch (error) {
      console.error("Error submitting rating:", error);
    } finally {
      submitting = false;
    }
  }

  function skipRating() {
    currentRatingIndex++;
    currentRating = 5; // Reset to default
    
    if (currentRatingIndex >= pendingRatings.length) {
      onClose();
    }
  }

  $: if (open) {
    loadPendingRatings();
  }

  $: currentPendingRating = pendingRatings[currentRatingIndex];
  $: hasRatings = pendingRatings.length > 0;
  $: remainingCount = pendingRatings.length - currentRatingIndex;
</script>

{#if open && hasRatings && currentPendingRating}
  <div class="overlay" on:click={(e) => { if (e.currentTarget === e.target) onClose(); }}>
    <div class="modal">
      <h3>Rate Your Viewing Experience</h3>
      
      {#if loading}
        <div class="loading">Loading your pending ratings...</div>
      {:else}
        <div class="rating-info">
          <p class="media-info">
            <strong>Media:</strong> {currentPendingRating.mediaType} #{currentPendingRating.mediaId}
          </p>
          <p class="date-info">
            <strong>Watched:</strong> {new Date(currentPendingRating.startedAt).toLocaleDateString()}
          </p>
          {#if currentPendingRating.notes}
            <p class="notes"><strong>Notes:</strong> {currentPendingRating.notes}</p>
          {/if}
        </div>

        <div class="rating-section">
          <label for="rating">Your Rating (1-10):</label>
          <input 
            id="rating"
            type="range" 
            min="1" 
            max="10" 
            step="0.5"
            bind:value={currentRating}
            class="rating-slider"
          />
          <span class="rating-value">{currentRating}/10</span>
        </div>

        <div class="progress">
          Rating {currentRatingIndex + 1} of {pendingRatings.length}
          {#if remainingCount > 1}
            ({remainingCount - 1} more after this)
          {/if}
        </div>

        <div class="actions">
          <button on:click={skipRating} class="ghost">Skip</button>
          <button on:click={submitRating} disabled={submitting}>
            {submitting ? "Submitting..." : "Submit Rating"}
          </button>
        </div>
      {/if}
    </div>
  </div>
{:else if open && !loading && !hasRatings}
  <div class="overlay" on:click={(e) => { if (e.currentTarget === e.target) onClose(); }}>
    <div class="modal">
      <h3>No Pending Ratings</h3>
      <p>You don't have any viewings that need ratings right now.</p>
      <div class="actions">
        <button on:click={onClose}>Close</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.4);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal {
    width: min(520px, 92vw);
    background: #111;
    color: #eee;
    border-radius: 12px;
    padding: 24px;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.5);
  }

  .loading {
    text-align: center;
    padding: 20px;
    opacity: 0.8;
  }

  .rating-info {
    margin: 16px 0;
    padding: 16px;
    background: #1a1a1a;
    border-radius: 8px;
  }

  .rating-info p {
    margin: 8px 0;
  }

  .media-info, .date-info {
    font-size: 14px;
  }

  .notes {
    font-size: 13px;
    opacity: 0.9;
  }

  .rating-section {
    margin: 20px 0;
    text-align: center;
  }

  .rating-section label {
    display: block;
    margin-bottom: 12px;
    font-weight: 500;
  }

  .rating-slider {
    width: 100%;
    margin: 12px 0;
    height: 6px;
    background: #333;
    border-radius: 3px;
    outline: none;
    -webkit-appearance: none;
  }

  .rating-slider::-webkit-slider-thumb {
    -webkit-appearance: none;
    appearance: none;
    width: 20px;
    height: 20px;
    background: #2d6cdf;
    border-radius: 50%;
    cursor: pointer;
  }

  .rating-slider::-moz-range-thumb {
    width: 20px;
    height: 20px;
    background: #2d6cdf;
    border-radius: 50%;
    cursor: pointer;
    border: none;
  }

  .rating-value {
    font-size: 18px;
    font-weight: bold;
    color: #2d6cdf;
  }

  .progress {
    text-align: center;
    margin: 16px 0;
    font-size: 14px;
    opacity: 0.8;
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    margin-top: 20px;
  }

  button {
    padding: 10px 16px;
    border-radius: 8px;
    border: none;
    background: #2d6cdf;
    color: white;
    cursor: pointer;
    font-size: 14px;
  }

  button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  button.ghost {
    background: transparent;
    border: 1px solid #444;
    color: #ddd;
  }

  button:hover:not(:disabled) {
    opacity: 0.9;
  }
</style>
