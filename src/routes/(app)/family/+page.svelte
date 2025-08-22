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
	let expandedRatings: Set<string> = new Set();

	async function loadFamilyHistory() {
		try {
			loading = true;
			const response = await axios.get("/group/family-history");
			familyHistory = Array.isArray(response.data) ? response.data : [];
			
			// Debug: Log the data we received
			console.log("Family history data:", familyHistory);
			if (familyHistory.length > 0) {
				console.log("First item attendees:", familyHistory[0].attendees);
			}

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
		if (rating === undefined || rating === null) return "—";
		return rating.toFixed(1);
	}

	function goToContent(mediaType: string, mediaId: string) {
		goto(`/${mediaType}/${mediaId}`);
	}

	function toggleRatings(sessionId: string) {
		if (expandedRatings.has(sessionId)) {
			expandedRatings.delete(sessionId);
		} else {
			expandedRatings.add(sessionId);
		}
		expandedRatings = expandedRatings; // Trigger reactivity
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
	<!-- Header Section -->
	<div class="header">
		<div class="header-content">
			<div class="title-section">
				<Icon i="people" wh={28} />
				<h1>Family Viewing History</h1>
			</div>
			<p class="subtitle">See what your family has watched together</p>
		</div>
	</div>

	{#if loading}
		<div class="loading-container">
			<Spinner />
		</div>
	{:else if error}
		<div class="error-container">
			<Error error={error} pretty="Failed to load family history!" />
		</div>
	{:else if familyHistory.length === 0}
		<div class="empty-state">
			<div class="empty-icon">
				<Icon i="film" wh={48} />
			</div>
			<h2>No Family Viewings Yet</h2>
			<p>Start watching movies and TV shows together to see your family history here!</p>
		</div>
	{:else}
		<!-- Sticky Filter Controls -->
		<div class="filter-bar">
			<div class="filter-controls">
				<div class="filter-group">
					<label for="type-filter">Type</label>
					<select id="type-filter" bind:value={filterType} class="filter-select">
						<option value="all">All</option>
						<option value="movie">Movies</option>
						<option value="episode">TV Episodes</option>
					</select>
				</div>
				
				<div class="filter-group">
					<label for="sort-filter">Sort</label>
					<select id="sort-filter" bind:value={sortBy} class="filter-select">
						<option value="date">Date</option>
						<option value="rating">Rating</option>
						<option value="title">Title</option>
					</select>
				</div>
				
				<button 
					class="sort-toggle" 
					onclick={() => sortOrder = sortOrder === "desc" ? "asc" : "desc"}
					title={sortOrder === "desc" ? "Descending" : "Ascending"}
				>
					<Icon i={sortOrder === "desc" ? "arrow-down" : "arrow-up"} wh={16} />
				</button>
			</div>
			
			<div class="results-count">
				{filteredHistory.length} of {familyHistory.length} viewings
			</div>
		</div>

		<!-- Content Cards -->
		<div class="content-grid">
			{#each filteredHistory as item}
				{@const content = contentDetails[item.mediaId]}
				<div class="viewing-card" onclick={() => goToContent(item.mediaType, item.mediaId)}>
					<!-- Poster Section -->
					<div class="poster-section">
						{#if content?.poster_path}
							<img 
								src={`https://image.tmdb.org/t/p/w200${content.poster_path}`} 
								alt={content.title || content.name}
								class="poster-image"
							/>
						{:else}
							<div class="poster-placeholder">
								<Icon i="film" wh={24} />
							</div>
						{/if}
					</div>

					<!-- Content Section -->
					<div class="content-section">
						<!-- Header Row -->
						<div class="card-header">
							<h3 class="content-title">{content?.title || content?.name || "Unknown Title"}</h3>
							<div class="header-badges">
								<span class="type-badge">{item.mediaType}</span>
							</div>
						</div>

						<!-- Metadata Row -->
						<div class="metadata-row">
							<div class="date-time">
								<Icon i="calendar" wh={10} />
								{formatDate(item.startedAt)}
							</div>
							<div class="attendee-count">
								<Icon i="people" wh={10} />
								{item.attendeeCount} {item.attendeeCount === 1 ? "person" : "people"}
							</div>
						</div>

						<!-- Collapsible Ratings Section -->
						{#if item.attendees.length > 0}
							<div class="ratings-section">
								<button 
									class="ratings-toggle"
									onclick={(e) => { e.stopPropagation(); toggleRatings(item.sessionId); }}
									title={expandedRatings.has(item.sessionId) ? "Hide individual ratings" : "Show individual ratings"}
								>
									<div class="ratings-summary">
										<Icon i="star" wh={14} />
										<span class="average-rating">
											{formatRating(item.averageRating)} average
										</span>
										<span class="expand-hint">Click to {expandedRatings.has(item.sessionId) ? "hide" : "show"} details</span>
									</div>
									<div class="expand-indicator">
										<Icon 
											i={expandedRatings.has(item.sessionId) ? "chevron-up" : "chevron-down"} 
											wh={16} 
											class="expand-icon"
										/>
									</div>
								</button>

								{#if expandedRatings.has(item.sessionId)}
									<div class="ratings-details">
										<div class="attendees-list">
											{#each item.attendees as attendee}
												<div class="attendee-item">
													<span class="attendee-name">{attendee.username}</span>
													{#if attendee.rating !== undefined && attendee.rating !== null}
														<span class="attendee-rating">
															<Icon i="star" wh={10} />
															{formatRating(attendee.rating)}
														</span>
													{:else}
														<span class="no-rating">—</span>
													{/if}
												</div>
											{/each}
										</div>
									</div>
								{/if}
							</div>
						{/if}

						<!-- Notes Section -->
						{#if item.notes}
							<div class="notes-section">
								<Icon i="note" wh={12} />
								<span class="notes-text">{item.notes}</span>
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
		min-height: 100vh;
		background: var(--bg-color, #f8f9fa);
		font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
	}

	/* Header Section */
	.header {
		background: #111827;
		color: white;
		padding: 2rem 1rem;
		text-align: center;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);

		@media (min-width: 768px) {
			padding: 3rem 2rem;
		}
	}

	.header-content {
		max-width: 1200px;
		margin: 0 auto;
	}

	.title-section {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.75rem;
		margin-bottom: 0.5rem;

		h1 {
			font-size: 1.75rem;
			font-weight: 600;
			margin: 0;
			letter-spacing: -0.025em;

			@media (min-width: 768px) {
				font-size: 2.25rem;
			}
		}
	}

	.subtitle {
		font-size: 1rem;
		opacity: 0.9;
		margin: 0;
		font-weight: 400;
	}

	/* Loading & Error States */
	.loading-container,
	.error-container {
		display: flex;
		justify-content: center;
		align-items: center;
		min-height: 50vh;
		padding: 2rem;
	}

	/* Empty State */
	.empty-state {
		text-align: center;
		padding: 4rem 2rem;
		max-width: 500px;
		margin: 0 auto;

		.empty-icon {
			color: #6b7280;
			margin-bottom: 1.5rem;
		}

		h2 {
			font-size: 1.5rem;
			font-weight: 600;
			color: #f9fafb;
			margin: 0 0 0.75rem 0;
		}

		p {
			color: #9ca3af;
			font-size: 1rem;
			line-height: 1.5;
			margin: 0;
		}
	}

	/* Filter Bar */
	.filter-bar {
		position: sticky;
		top: 0;
		background: #1f2937;
		border-bottom: 1px solid #374151;
		padding: 1rem;
		z-index: 10;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);

		@media (min-width: 768px) {
			padding: 1.25rem 2rem;
		}
	}

	.filter-controls {
		display: flex;
		align-items: center;
		gap: 1rem;
		margin-bottom: 0.75rem;

		@media (min-width: 768px) {
			margin-bottom: 0;
		}
	}

	.filter-group {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;

		label {
			font-size: 0.75rem;
			font-weight: 500;
			color: #9ca3af;
			text-transform: uppercase;
			letter-spacing: 0.05em;
		}
	}

	.filter-select {
		padding: 0.5rem 0.75rem;
		border: 1px solid #4b5563;
		border-radius: 0.5rem;
		background: #374151;
		font-size: 0.875rem;
		font-weight: 500;
		color: #f9fafb;
		cursor: pointer;
		transition: all 0.2s ease;

		&:focus {
			outline: none;
			border-color: #3b82f6;
			box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
		}

		&:hover {
			border-color: #6b7280;
		}
	}

	.sort-toggle {
		padding: 0.5rem;
		border: 1px solid #4b5563;
		border-radius: 0.5rem;
		background: #374151;
		cursor: pointer;
		transition: all 0.2s ease;
		display: flex;
		align-items: center;
		justify-content: center;
		color: #f9fafb;

		&:hover {
			background: #4b5563;
			border-color: #6b7280;
		}

		&:focus {
			outline: none;
			border-color: #3b82f6;
			box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
		}
	}

	.results-count {
		font-size: 0.875rem;
		color: #9ca3af;
		font-weight: 500;
	}

	/* Content Grid */
	.content-grid {
		max-width: 1200px;
		margin: 0 auto;
		padding: 1rem;
		display: flex;
		flex-direction: column;
		gap: 1rem;

		@media (min-width: 768px) {
			padding: 2rem;
			gap: 1.5rem;
		}
	}

	/* Viewing Card */
	.viewing-card {
		background: #1f2937;
		border-radius: 1rem;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
		overflow: hidden;
		cursor: pointer;
		transition: all 0.2s ease;
		border: 1px solid #374151;

		&:hover {
			transform: translateY(-2px);
			box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
		}

		@media (min-width: 768px) {
			display: flex;
		}
	}

	/* Poster Section */
	.poster-section {
		flex-shrink: 0;
		width: 100%;
		height: 200px;
		background: #374151;

		@media (min-width: 768px) {
			width: 140px;
			height: 210px;
		}

		.poster-image {
			width: 100%;
			height: 100%;
			object-fit: cover;
		}

		.poster-placeholder {
			width: 100%;
			height: 100%;
			display: flex;
			align-items: center;
			justify-content: center;
			color: #6b7280;
			background: #374151;
		}
	}

	/* Content Section */
	.content-section {
		flex: 1;
		padding: 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;

		@media (min-width: 768px) {
			padding: 1.25rem;
		}
	}

	/* Card Header */
	.card-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: 0.75rem;
	}

	.content-title {
		font-size: 1.125rem;
		font-weight: 600;
		color: #f9fafb;
		margin: 0;
		line-height: 1.3;
		flex: 1;

		@media (min-width: 768px) {
			font-size: 1.25rem;
		}
	}

	.header-badges {
		display: flex;
		gap: 0.5rem;
		flex-shrink: 0;
	}

	.type-badge {
		background: #3b82f6;
		color: white;
		padding: 0.25rem 0.5rem;
		border-radius: 0.375rem;
		font-size: 0.75rem;
		font-weight: 500;
		text-transform: capitalize;
	}

	/* Metadata Row */
	.metadata-row {
		display: flex;
		gap: 1rem;
		font-size: 0.875rem;
		color: #9ca3af;
	}

	.date-time,
	.attendee-count {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		font-weight: 500;
	}

	/* Ratings Section */
	.ratings-section {
		border-top: 1px solid #374151;
		padding-top: 0.75rem;
	}

	.ratings-toggle {
		width: 100%;
		background: none;
		border: none;
		padding: 0.75rem;
		cursor: pointer;
		display: flex;
		justify-content: space-between;
		align-items: center;
		border-radius: 0.5rem;
		transition: all 0.2s ease;
		border: 1px solid transparent;

		&:hover {
			background: #374151;
			border-color: #4b5563;
		}

		&:focus {
			outline: none;
			background: #374151;
			border-color: #3b82f6;
		}
	}

	.ratings-summary {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		color: #fbbf24;
		font-weight: 600;
		font-size: 0.875rem;
		flex: 1;
	}

	.average-rating {
		color: #fbbf24;
	}

	.expand-hint {
		color: #9ca3af;
		font-weight: 400;
		font-size: 0.75rem;
		margin-left: 0.5rem;
		opacity: 0.8;
	}

	.expand-indicator {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		border-radius: 50%;
		background: #374151;
		transition: all 0.2s ease;
	}

	.expand-icon {
		color: #9ca3af;
		transition: transform 0.2s ease;
		font-size: 0.875rem;
	}

	.ratings-toggle:hover .expand-indicator {
		background: #4b5563;
	}

	.ratings-toggle:hover .expand-icon {
		color: #d1d5db;
	}

	.ratings-details {
		margin-top: 0.75rem;
		padding-top: 0.75rem;
		border-top: 1px solid #374151;
	}

	.attendees-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.attendee-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.5rem;
		background: #374151;
		border-radius: 0.5rem;
		font-size: 0.875rem;
	}

	.attendee-name {
		font-weight: 500;
		color: #e5e7eb;
	}

	.attendee-rating {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		color: #fbbf24;
		font-weight: 600;
		font-size: 0.75rem;
	}

	.no-rating {
		color: #9ca3af;
		font-size: 0.75rem;
	}

	/* Notes Section */
	.notes-section {
		display: flex;
		align-items: flex-start;
		gap: 0.5rem;
		padding: 0.75rem;
		background: #1e293b;
		border-radius: 0.5rem;
		border-left: 3px solid #f59e0b;
		font-size: 0.875rem;
		color: #fbbf24;
	}

	.notes-text {
		font-style: italic;
		line-height: 1.4;
	}

	/* Dark Theme Support */
	@media (prefers-color-scheme: dark) {
		.family-history {
			background: #111827;
		}

		.filter-bar {
			background: #1f2937;
			border-bottom-color: #374151;
		}

		.viewing-card {
			background: #1f2937;
			border-color: #374151;
		}

		.content-title {
			color: #f9fafb;
		}

		.attendee-item {
			background: #374151;
		}

		.attendee-name {
			color: #e5e7eb;
		}

		.filter-select,
		.sort-toggle {
			background: #374151;
			border-color: #4b5563;
			color: #f9fafb;

			&:hover {
				background: #4b5563;
			}
		}

		.poster-placeholder {
			background: #374151;
			color: #6b7280;
		}

		.ratings-toggle:hover {
			background: #374151;
		}

		.ratings-toggle:focus {
			background: #4b5563;
		}
	}
</style>

