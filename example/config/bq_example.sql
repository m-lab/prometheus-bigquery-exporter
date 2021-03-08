-- Example query.
WITH example_data as (
    SELECT "a" as label, 5 as widgets
    UNION ALL
    SELECT "b" as label, 2 as widgets
    UNION ALL
    SELECT "b" as label, 3 as widgets
)

SELECT
   label, SUM(widgets) as value
FROM
   example_data
GROUP BY
   label
