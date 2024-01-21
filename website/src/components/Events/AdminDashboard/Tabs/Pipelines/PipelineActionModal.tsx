import React, { useState } from "react";
import { PipelineAction, PipelineEvent } from "@/types/models/Pipeline";
import { FieldValue, FormField, FormOptionCustomLabelValue, FormStructure } from "@/types/models/Form";
import FormBuilder from "@/components/Form/FormBuilder";
import Select from "@/components/Form/inputs/Select";
import { toTitleCase } from "@/utils/strings";

interface PipelineActionModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSelect: (action: PipelineAction | PipelineEvent) => void;
  modalType: "action" | "event";
  eventForms?: FormStructure[];
}

const PipelineActionModal: React.FC<PipelineActionModalProps> = ({
  isOpen,
  onClose,
  onSelect,
  modalType,
  eventForms
}) => {
  const options =
    modalType === "action"
      ? ["SendEmail", "AllowFormAccess", "Webhook"]
      : ["FormSubmission", "FieldChange"];
  const [selectedType, setSelectedType] = useState<string>();

  const createEventObject = (
    formData: Record<string, any>
  ): PipelineEvent | null => {
    switch (selectedType) {
      case "FormSubmission":
        return {
          name: formData.name,
          type: "FormSubmission",
          formSubmission: {
            onFormID: formData.onFormID,
          },
        };
      case "FieldChange":
        return {
          name: formData.name,
          type: "FieldChange",
          fieldChange: {
            onFormID: formData.onFormID,
            onFieldID: formData.onFieldID,
            condition: {
                comparison: "eq", // dropdown
                value: "accepted", // todo: make these out
            },
          },
        };
      default:
        return null;
    }
  };

  const createActionObject = (
    formData: Record<string, any>
  ): PipelineAction | null => {
    switch (selectedType) {
      case "SendEmail":
        return {
          name: formData.name,
          type: "SendEmail",
          sendEmail: {
            emailTemplateID: formData.emailTemplateID,
            emailFieldID: formData.emailFieldID,
          },
        };
      case "AllowFormAccess":
        return {
          name: formData.name,
          type: "AllowFormAccess",
          allowFormAccess: {
            toFormID: formData.toFormID,
            options: {
              expiration: {
                inHoursFromPipelineRun: formData.expiration,
              },
            },
          },
        };
      case "Webhook":
        return {
          name: formData.name,
          type: "Webhook",
          webhook: {
            url: formData.url,
            method: formData.method,
            headers: formData.headers, // Ensure headers are correctly handled
          },
        };
      default:
        return null;
    }
  };

  const handleAddAction = (formData: Record<string, any>) => {
    if (selectedType) {
      const action = createActionObject(formData);
      if (action) {
        onSelect(action);
        onClose();
      }
    }
  };

  const handleAddEvent = (formData: Record<string, any>) => {
    if (selectedType) {
      const event = createEventObject(formData);
      if (event) {
        onSelect(event);
        onClose();
      }
    }
  }

  const renderPipelineEventForm = (t: string | undefined) => {
    if (!t) return null;

    let formStructure: FormStructure | null = null;

    switch (t) {
      case "SendEmail":
        formStructure = createSendEmailFormStructure(eventForms);
        break;
      case "AllowFormAccess":
        formStructure = createAllowFormAccessFormStructure(eventForms);
        break;
      case "Webhook":
        formStructure = createWebhookFormStructure(eventForms);
        break;
      case "FormSubmission":
        formStructure = createFormSubmissionFormStructure(eventForms);
        break;
      case "FieldChange":
        formStructure = createFieldChangeFormStructure(eventForms);
        break;
      default:
        return null;
    }

    if (formStructure) {
      return (
        <FormBuilder
          formStructure={formStructure}
          submissionFunction={modalType === "action" ? handleAddAction : handleAddEvent}
          buttonText={modalType === "action" ? "Add Action" : "Set Event"}
        />
      );
    }

    return null;
  };

  return (
    <>
      {isOpen && (
        <div className="modal modal-open">
          <div className="modal-box">
            <h3 className="text-xl font-bold">Add a new {modalType}</h3>

            <Select
              field={
                {
                  question: `${toTitleCase(modalType)} Type`,
                  type: "select",
                  key: "type",
                  options: options,
                } as FormField
              }
              onChange={(k, v: FieldValue) => {
                setSelectedType(v as string)
              }}
              isMultiSelect={false}
              allowArbitraryInput={false}
            />

            {modalType !== null && renderPipelineEventForm(selectedType)}
            <div className="modal-action">
              <button onClick={() => {
                setSelectedType(undefined);
                onClose();
              }} className="btn btn-outline">
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
};

export default PipelineActionModal;

// Helper functions to create form structures for each action type
const createSendEmailFormStructure = (eventForms: FormStructure[] | undefined): FormStructure => {
  return {
    attrs: [
      {
        question: "Name This Action",
        type: "text",
        key: "name",
        required: true,
      },
      {
        question: "Email Template ID",
        type: "text",
        key: "emailTemplateID",
        required: true,
      },
      {
        question: "Email Field ID",
        type: "text",
        key: "emailFieldID",
        required: true,
      },
      // Add more fields as needed
    ],
  };
};

const createAllowFormAccessFormStructure = (eventForms: FormStructure[] | undefined): FormStructure => {
  return {
    attrs: [
      {
        question: "Name This Action",
        type: "text",
        key: "name",
        required: true,
      },
      {
        question: "To Form",
        type: "select",
        key: "toFormID",
        required: true,
        options: eventForms?.map((form) => {
          return {
            value: form.id,
            label: `${form.name} (${form.id})`
          } as FormOptionCustomLabelValue;
        })
      },
      {
        question: "Expiration (in hours)",
        type: "number",
        key: "expiration",
        additionalValidation: { min: 1 },
      },
      // Add more fields as needed
    ],
  };
};

const createWebhookFormStructure = (eventForms: FormStructure[] | undefined): FormStructure => {
  return {
    attrs: [
      {
        question: "Name This Action",
        type: "text",
        key: "name",
        required: true,
      },
      {
        question: "URL",
        type: "text",
        key: "url",
        required: true,
      },
      {
        question: "Method",
        type: "select",
        key: "method",
        options: ["POST", "GET", "PUT", "DELETE"],
        required: true,
      },
      // Add fields for headers and body as necessary
    ],
  };
};

const createFormSubmissionFormStructure = (eventForms: FormStructure[] | undefined): FormStructure => {
  return {
    attrs: [
      {
        question: "Name This Action",
        type: "text",
        key: "name",
        required: true,
      },
      {
        question: "On Form",
        type: "select",
        key: "onFormID",
        required: true,
        options: eventForms?.map((form) => {
          return {
            value: form.id,
            label: `${form.name} (${form.id})`
          } as FormOptionCustomLabelValue;
        })
      },
    ],
  };
};

const createFieldChangeFormStructure = (eventForms: FormStructure[] | undefined): FormStructure => {
  return {
    attrs: [
      {
        question: "Name This Action",
        type: "text",
        key: "name",
        required: true,
      },
      {
        question: "On Form",
        type: "text",
        key: "onFormID",
        required: true,
        options: eventForms?.map((form) => {
          return {
            value: form.id,
            label: `${form.name} (${form.id})`
          } as FormOptionCustomLabelValue;
        })
      },
      {
        question: "On Field ID",
        type: "text",
        key: "onFieldID",
        required: true,
      },
      // TODO: Need to do the condition
    ],
  };
};
