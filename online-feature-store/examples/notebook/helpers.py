import re
from cloudpathlib import CloudPath


def get_latest_path(root_folder: str, partition_col: str = "ts", folder_type: str = "parquet") -> str:
    """
    Returns the path including the latest timestamp from the given directory location in cloud storage
    Args:
            root_folder (str): The root path of cloud storage folder (GCS/S3/Azure) which has multiple timestamp folders in it
            partition_col (str): The partition column name to search for timestamps (default: "ts")
            folder_type (str): Type of folder - parquet or delta (default: "parquet")
    Returns:
            str. Returns the cloud storage path corresponding to the latest timestamp in the root folder.
    Raises:
            FileNotFoundError. Raises the exception if no valid folders are found for the given date_str.
    """

    # CloudPath supports GCS, S3 and Azure paths
    # For GCS paths starting with gs://
    # For S3 paths starting with s3://
    # For Azure paths starting with az://
    subdirectories = [str(x) for x in CloudPath(root_folder).iterdir()]

    paths_with_time = []

    for subdirectory in subdirectories:
        try:
            # Extract timestamp from partition column
            paths_with_time.append(
                (
                    f"{subdirectory}",
                    re.search(f"{partition_col}=(.*)$", subdirectory).group(1),
                )
            )
        except:
            pass

    if not paths_with_time:
        raise FileNotFoundError(
            f"No folders found with partition column {partition_col}"
        )

    # Sort by timestamp in descending order
    paths_with_time.sort(key=lambda t: t[1], reverse=True)

    if folder_type == "parquet":
        # For parquet, find first path that has _SUCCESS file
        for path, _ in paths_with_time:
            success_file = path + "_SUCCESS"
            if CloudPath(success_file).is_file():
                print(f"Returning path: {path}")
                return path
        raise FileNotFoundError("No folder with _SUCCESS file found")
    else:
        # For other types, just return latest path
        latest_path = paths_with_time[0][0]
        print(f"Returning path: {latest_path}")
        return latest_path
