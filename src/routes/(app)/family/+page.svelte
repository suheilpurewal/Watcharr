<script lang="ts">
	import { onMount } from "svelte";
	import axios from "axios";
	import Spinner from "@/lib/Spinner.svelte";
	import Error from "@/lib/Error.svelte";
	import Icon from "@/lib/Icon.svelte";
	import { goto } from "$app/navigation";

	type FamilyHistoryItem = {
		sessionId: string;
		mediaId: string;
		mediaType: "movie" | "episode";
		startedAt: string;
		notes?: string;
		attendeeCount: number;
		averageRating?: number;
		attendees: Array<{
			userId: number;
			username: string;
			rating?: number;
		}>;
	};

	let familyHistory: FamilyHistoryItem[] = [];
	let filteredHistory: FamilyHistoryItem[] = [];
	let loading = true;
	let error: Error | undefined;
	let contentDetails: Record<string, any> = {};
	let filterType: "all" | "movie" | "episode" = "all";
	let sortBy: "date" | "rating" | "title" = "date";
	let sortOrder: "desc" | "asc" = "desc";

	async function loadFamilyHistory() {
		try {
			loading = true;
			const response = await axios.get("/api/group/family-history");
			familyHistory = Array.isArray(response.data) ? response.data : [];

			// Load content details for each item
			for (const item of familyHistory) {
				if (!contentDetails[item.mediaId]) {
					try {
						const contentResponse = await axios.get(
							`/content/${item.mediaType}/${item.mediaId}`
						);
						contentDetails[item.mediaId] = contentResponse.data;
					} catch (err) {
						console.warn(`Failed to load details for ${item.mediaType} ${item.mediaId}:`, err);
						contentDetails[item.mediaId] = {
							title: item.mediaType === "movie" ? "Unknown Movie" : "Unknown Episode",
							poster_path: null
						};
					}
				}
			}
		} catch (err) {
			console.error("Failed to load family history:", err);
			error = err as Error;
		} finally {
			loading = false;
		}
	}

	function formatDate(dateString: string): string {
		const date = new Date(dateString);
		return date.toLocaleDateString("en-US", {
			year: "numeric",
			month: "short",
			day: "numeric",
			hour: "2-digit",
			minute: "2-digit"
		});
	}

	function formatRating(rating?: number): string {
		if (!rating) return "—";
		return rating.toFixed(1);
	}

	function goToContent(mediaType: string, mediaId: string) {
		goto(`/${mediaType}/${mediaId}`);
	}

	function applyFiltersAndSort() {
		if (!Array.isArray(familyHistory)) {
			filteredHistory = [];
			return;
		}
		
		let filtered = [...familyHistory];

		// Apply type filter
		if (filterType !== "all") {
			filtered = filtered.filter(item => item.mediaType === filterType);
		}

		// Apply sorting
		filtered.sort((a, b) => {
			let comparison = 0;
			
			switch (sortBy) {
				case "date":
					comparison = new Date(a.startedAt).getTime() - new Date(b.startedAt).getTime();
					break;
				case "rating":
					const aRating = a.averageRating || 0;
					const bRating = b.averageRating || 0;
					comparison = aRating - bRating;
					break;
				case "title":
					const aTitle = contentDetails[a.mediaId]?.title || contentDetails[a.mediaId]?.name || "";
					const bTitle = contentDetails[b.mediaId]?.title || contentDetails[b.mediaId]?.name || "";
					comparison = aTitle.localeCompare(bTitle);
					break;
			}

			return sortOrder === "desc" ? -comparison : comparison;
		});

		filteredHistory = filtered;
	}

	// Reactive statement to update filtered history when filters change
	$: if (Array.isArray(familyHistory) && familyHistory.length > 0 && Object.keys(contentDetails).length > 0) {
		applyFiltersAndSort();
	}

	onMount(() => {
		loadFamilyHistory();
	});
</script>

<svelte:head>
	<title>Family History - Watcharr</title>
</svelte:head>

<div class="family-history">
	<div class="header">
		<h1>
			<Icon i="people" wh={32} />
			Family Viewing History
		</h1>
		<p>See what your family has watched together</p>
	</div>

	{#if loading}
		<Spinner />
	{:else if error}
		<Error error={error} pretty="Failed to load family history!" />
	{:else if familyHistory.length === 0}
		<div class="empty-state">
			<Icon i="film" wh={64} />
			<h2>No Family Viewings Yet</h2>
			<p>Start watching movies and TV shows together to see your family history here!</p>
		</div>
	{:else}
		<!-- Filter and Sort Controls -->
		<div class="controls">
			<div class="filter-controls">
				<label>
					<Icon i="filter" wh={16} />
					Filter by type:
					<select bind:value={filterType}>
						<option value="all">All</option>
						<option value="movie">Movies</option>
						<option value="episode">TV Episodes</option>
					</select>
				</label>
			</div>
			
			<div class="sort-controls">
				<label>
					<Icon i="sort" wh={16} />
					Sort by:
					<select bind:value={sortBy}>
						<option value="date">Date</option>
						<option value="rating">Rating</option>
						<option value="title">Title</option>
					</select>
				</label>
				
				<button 
					class="sort-order" 
					onclick={() => sortOrder = sortOrder === "desc" ? "asc" : "desc"}
					title={sortOrder === "desc" ? "Descending" : "Ascending"}
				>
					<Icon i={sortOrder === "desc" ? "arrow-down" : "arrow-up"} wh={16} />
				</button>
			</div>
		</div>

		<!-- Results Summary -->
		<div class="results-summary">
			Showing {filteredHistory.length} of {familyHistory.length} viewings
		</div>

		<div class="history-list">
			{#each filteredHistory as item}
				{@const content = contentDetails[item.mediaId]}
				<div class="history-item" onclick={() => goToContent(item.mediaType, item.mediaId)}>
					<div class="poster">
						{#if content?.poster_path}
							<img 
								src={`https://image.tmdb.org/t/p/w200${content.poster_path}`} 
								alt={content.title || content.name}
							/>
						{:else}
							<div class="poster-placeholder">
								<Icon i="film" wh={32} />
							</div>
						{/if}
					</div>

					<div class="details">
						<div class="title-section">
							<h3>{content?.title || content?.name || "Unknown Title"}</h3>
							<div class="metadata">
								<span class="date">{formatDate(item.startedAt)}</span>
								<span class="type">{item.mediaType}</span>
								{#if item.averageRating}
									<span class="rating">
										<Icon i="star" wh={14} />
										{formatRating(item.averageRating)}
									</span>
								{/if}
							</div>
						</div>

						<div class="attendees">
							<div class="attendee-count">
								<Icon i="people" wh={16} />
								{item.attendeeCount} {item.attendeeCount === 1 ? "person" : "people"}
							</div>
							<div class="attendee-list">
								{#each item.attendees as attendee}
									<div class="attendee">
										<span class="name">{attendee.username}</span>
										{#if attendee.rating}
											<span class="individual-rating">
												<Icon i="star" wh={12} />
												{formatRating(attendee.rating)}
											</span>
										{:else}
											<span class="no-rating">—</span>
										{/if}
									</div>
								{/each}
							</div>
						</div>

						{#if item.notes}
							<div class="notes">
								<Icon i="note" wh={14} />
								{item.notes}
							</div>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<style lang="scss">
	.family-history {
		max-width: 1200px;
		margin: 0 auto;
		padding: 20px;
	}

	.header {
		text-align: center;
		margin-bottom: 40px;

		h1 {
			display: flex;
			align-items: center;
			justify-content: center;
			gap: 12px;
			font-size: 2.5rem;
			margin-bottom: 8px;
		}

		p {
			color: #666;
			font-size: 1.1rem;
		}
	}

	.empty-state {
		text-align: center;
		padding: 60px 20px;
		color: #666;

		h2 {
			margin: 20px 0 10px;
			color: #333;
		}

		p {
			font-size: 1.1rem;
		}
	}

	.controls {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 20px;
		padding: 20px;
		background: #f8f9fa;
		border-radius: 12px;
		gap: 20px;

		@media (max-width: 768px) {
			flex-direction: column;
			gap: 15px;
		}

		.filter-controls,
		.sort-controls {
			display: flex;
			align-items: center;
			gap: 12px;

			label {
				display: flex;
				align-items: center;
				gap: 8px;
				font-weight: 500;
				color: #555;
			}

			select {
				padding: 8px 12px;
				border: 1px solid #ddd;
				border-radius: 6px;
				background: white;
				font-size: 0.9rem;
				cursor: pointer;

				&:focus {
					outline: none;
					border-color: #007bff;
					box-shadow: 0 0 0 2px rgba(0, 123, 255, 0.25);
				}
			}
		}

		.sort-order {
			padding: 8px;
			border: 1px solid #ddd;
			border-radius: 6px;
			background: white;
			cursor: pointer;
			transition: all 0.2s ease;

			&:hover {
				background: #e9ecef;
				border-color: #007bff;
			}

			&:focus {
				outline: none;
				border-color: #007bff;
				box-shadow: 0 0 0 2px rgba(0, 123, 255, 0.25);
			}
		}
	}

	.results-summary {
		text-align: center;
		margin-bottom: 20px;
		color: #666;
		font-size: 0.9rem;
		font-style: italic;
	}

	.history-list {
		display: flex;
		flex-direction: column;
		gap: 20px;
	}

	.history-item {
		display: flex;
		gap: 20px;
		padding: 20px;
		background: #f8f9fa;
		border-radius: 12px;
		cursor: pointer;
		transition: all 0.2s ease;
		border: 2px solid transparent;

		&:hover {
			background: #e9ecef;
			border-color: #007bff;
			transform: translateY(-2px);
			box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
		}

		@media (max-width: 768px) {
			flex-direction: column;
			gap: 15px;
		}
	}

	.poster {
		flex-shrink: 0;
		width: 120px;
		height: 180px;
		border-radius: 8px;
		overflow: hidden;

		img {
			width: 100%;
			height: 100%;
			object-fit: cover;
		}

		.poster-placeholder {
			width: 100%;
			height: 100%;
			background: #ddd;
			display: flex;
			align-items: center;
			justify-content: center;
			color: #999;
		}

		@media (max-width: 768px) {
			width: 80px;
			height: 120px;
			align-self: center;
		}
	}

	.details {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 15px;
	}

	.title-section {
		h3 {
			font-size: 1.4rem;
			margin-bottom: 8px;
			color: #333;
		}

		.metadata {
			display: flex;
			gap: 15px;
			flex-wrap: wrap;
			font-size: 0.9rem;
			color: #666;

			.date {
				font-weight: 500;
			}

			.type {
				text-transform: capitalize;
				background: #007bff;
				color: white;
				padding: 2px 8px;
				border-radius: 4px;
				font-size: 0.8rem;
			}

			.rating {
				display: flex;
				align-items: center;
				gap: 4px;
				color: #ffc107;
				font-weight: 500;
			}
		}
	}

	.attendees {
		.attendee-count {
			display: flex;
			align-items: center;
			gap: 6px;
			font-weight: 500;
			color: #555;
			margin-bottom: 8px;
		}

		.attendee-list {
			display: flex;
			flex-wrap: wrap;
			gap: 12px;
		}

		.attendee {
			display: flex;
			align-items: center;
			gap: 6px;
			background: white;
			padding: 6px 12px;
			border-radius: 20px;
			font-size: 0.9rem;
			border: 1px solid #ddd;

			.name {
				font-weight: 500;
			}

			.individual-rating {
				display: flex;
				align-items: center;
				gap: 2px;
				color: #ffc107;
				font-size: 0.8rem;
			}

			.no-rating {
				color: #999;
				font-size: 0.8rem;
			}
		}
	}

	.notes {
		display: flex;
		align-items: flex-start;
		gap: 8px;
		padding: 12px;
		background: white;
		border-radius: 8px;
		border-left: 4px solid #007bff;
		font-style: italic;
		color: #555;
	}

	@media (max-width: 768px) {
		.family-history {
			padding: 15px;
		}

		.header h1 {
			font-size: 2rem;
		}

		.history-item {
			padding: 15px;
		}

		.attendees .attendee-list {
			flex-direction: column;
			gap: 8px;
		}
	}
</style>
