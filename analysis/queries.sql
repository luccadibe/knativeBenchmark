
-- Count the number of events that match the expected number of triggers
SELECT 
    ex.id AS experiment_id,
    ex.triggers AS expected_triggers,
    COUNT(counts.event_id) AS total_events,
    SUM(CASE WHEN counts.event_count = ex.triggers THEN 1 ELSE 0 END) AS matching_events,
    SUM(CASE WHEN counts.event_count != ex.triggers THEN 1 ELSE 0 END) AS non_matching_events
FROM 
    (
        SELECT 
            e.event_id,
            r.experiment_id,
            COUNT(e.event_id) AS event_count
        FROM 
            events e
        JOIN 
            requests r ON e.event_id = r.event_id
        GROUP BY 
            e.event_id, r.experiment_id
    ) AS counts
JOIN 
    experiments ex ON counts.experiment_id = ex.id
GROUP BY 
    ex.id, ex.triggers;

-- With average events recieved

SELECT 
    ex.id AS experiment_id,
    ex.triggers AS expected_arrived_events,
    COUNT(counts.event_id) AS total_events,
    SUM(CASE WHEN counts.event_count = ex.triggers THEN 1 ELSE 0 END) AS matching_events,
    SUM(CASE WHEN counts.event_count != ex.triggers THEN 1 ELSE 0 END) AS non_matching_events,
    AVG(counts.event_count) AS avg_arrived_events_per_event_id
FROM 
    (
        SELECT 
            e.event_id,
            r.experiment_id,
            COUNT(e.event_id) AS event_count
        FROM 
            events e
        JOIN 
            requests r ON e.event_id = r.event_id
        GROUP BY 
            e.event_id, r.experiment_id
    ) AS counts
JOIN 
    experiments ex ON counts.experiment_id = ex.id
GROUP BY 
    ex.id, ex.triggers;


-- avg time to process event

SELECT 
    r.experiment_id,
    ex.triggers,
    COUNT(e.event_id) AS total_events,
    AVG(
        CAST(strftime('%s', e.timestamp) AS REAL) - 
        CAST(strftime('%s', r.timestamp) AS REAL)
    ) AS avg_processing_time_seconds
FROM 
    events e
JOIN 
    requests r ON e.event_id = r.event_id
JOIN 
    experiments ex ON r.experiment_id = ex.id
GROUP BY 
    r.experiment_id;


-- compare timestamps of slowest events and get the rps of the corresponding experiment

SELECT 
    r.event_id,
    r.timestamp AS request_timestamp,
    e.timestamp AS event_timestamp,
    CAST(strftime('%s', e.timestamp) AS REAL) - CAST(strftime('%s', r.timestamp) AS REAL) AS processing_time_seconds,
    ex.rps AS experiment_rps
FROM 
    requests r
JOIN 
    events e ON r.event_id = e.event_id
JOIN 
    experiments ex ON r.experiment_id = ex.id
ORDER BY 
    processing_time_seconds DESC
LIMIT 10;

-- were all events processed?

SELECT 
    ex.id AS experiment_id,
    ex.rps AS requests_per_second,
    COUNT(r.event_id) AS total_requests,
    SUM(CASE WHEN e.event_id IS NOT NULL THEN 1 ELSE 0 END) AS matched_events,
    SUM(CASE WHEN e.event_id IS NULL THEN 1 ELSE 0 END) AS unmatched_requests
FROM 
    requests r
JOIN 
    experiments ex ON r.experiment_id = ex.id
LEFT JOIN 
    events e ON r.event_id = e.event_id
GROUP BY 
    ex.id, ex.rps;

/* 
result for preliminary run:
1|1000|291709|0|291709
2|100|30000|30000|0
3|200|59990|59990|0
4|300|89965|89965|0
5|400|119934|119933|1
7|500|149909|149908|1
8|600|179844|179843|1
9|700|209822|209821|1
10|800|239770|114614|125156
11|900|269679|0|269679 */

