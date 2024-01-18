import FormCreator from "@/components/Form/Creator/FormCreator";
import FormBuilder from "@/components/Form/FormBuilder";
import { eventEmitter } from "@/events/EventEmitter";
import { updateForm } from "@/services/FormService";
import { EventModel } from "@/types/models/Event";
import { FormStructure } from "@/types/models/Form";
import { useState } from "react";
import { ToastType, useToast } from "@/components/Toast/ToastContext";
import LinkIcon from "@/components/Icons/LinkIcon";
import { PipelineConfiguration } from "@/types/models/Pipeline";
import { UpdatePipeline } from "@/services/PipelineService";
import PipelineSettings from "./PipelineSettings";

interface SelectPipelineProps {
  pipeline: PipelineConfiguration;
  onDelete: () => void;
}

const SelectPipeline: React.FC<SelectPipelineProps> = ({
  pipeline,
  onDelete,
}) => {
  const [pageSelected, setPageSelected] = useState<
    "view" | "edit" | "settings" | "runs"
  >("view");
  const [pipelineConfig, setPipeline] =
    useState<PipelineConfiguration>(pipeline);
  const { showToast } = useToast();

  // Edit
  const onFormStructureChange = (newPipelineConfig: PipelineConfiguration) => {
    UpdatePipeline(newPipelineConfig)
      .then(() => {
        eventEmitter.emit("success", "Successfully updated form!");
        setPipeline(newPipelineConfig);
      })
      .catch((err) => {});
  };

  const changePipeline = (pipeline: PipelineConfiguration) => {
    setPipeline(pipeline);
  };

  const isActive = (page: string) =>
    page === pageSelected ? "btn-active" : "";

  return (
    <>
      <div className="flex space-x-2 bg-gray-100 p-2 rounded">
        <button
          className={`btn ${isActive("view")}`}
          onClick={() => setPageSelected("view")}
        >
          View
        </button>
        <button
          className={`btn ${isActive("edit")}`}
          onClick={() => setPageSelected("edit")}
        >
          Edit
        </button>
        <button
          className={`btn ${isActive("settings")}`}
          onClick={() => setPageSelected("settings")}
        >
          Settings
        </button>
        <button
          className={`btn ${isActive("runs")}`}
          onClick={() => setPageSelected("runs")}
        >
          Runs
        </button>
      </div>

      <h2 className="text-2xl font-semibold text-gray-800 mt-4 mb-2">
        {pipeline.name}
      </h2>

      {pageSelected === "edit" && <p>Edit Pipeline</p>}

      {pageSelected === "view" && <p>View Pipeline</p>}

      {pageSelected === "settings" && (
        <PipelineSettings pipeline={pipeline} onDelete={onDelete} changePipeline={changePipeline} />
      )}

      {pageSelected === "runs" && <p>Runs</p>}

      {pageSelected !== "edit" &&
        pageSelected !== "runs" &&
        pageSelected !== "view" &&
        pageSelected !== "settings" && <p>Could not find selected page.</p>}
    </>
  );
};

export default SelectPipeline;