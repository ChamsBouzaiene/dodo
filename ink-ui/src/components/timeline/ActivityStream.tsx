import React from "react";
import { Box } from "ink";
import { ActivityItem } from "./ActivityItem.js";
import { CodeDiff } from "./CodeDiff.js";
import type { Activity } from "../../types.js";

/**
 * Props for the ActivityStream component.
 */
type ActivityStreamProps = {
  /** Array of activities to display */
  activities: Activity[];
};

export const ActivityStream: React.FC<ActivityStreamProps> = ({ activities }) => {
  if (activities.length === 0) {
    return null;
  }

  return (
    <Box flexDirection="column" marginLeft={2}>
      {activities.map((activity) => (
        <Box key={activity.id} flexDirection="column">
          <ActivityItem activity={activity} />
          {activity.codeChange && <CodeDiff codeChange={activity.codeChange} />}
        </Box>
      ))}
    </Box>
  );
};


