package handlers

import (
	"assignment-2/internal/models"
	"assignment-2/internal/utility"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
	"unicode"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// Handler holding shared dependencies and local registration counter.
type Handler struct {
	Client            *firestore.Client
	RegistrationCount atomic.Int64
	WebhookCount      atomic.Int64
}

// addRegistrationDocImpl stores a registration document, returns a generated ID.
// Defined as variable to allow for replacement in tests.
var addRegistrationDocImpl = func(ctx context.Context, client *firestore.Client, reg map[string]any) (string, error) {
	ref, _, err := client.Collection(utility.RegistrationsCollection).Add(ctx, reg)
	if err != nil {
		return "", err
	}
	return ref.ID, nil
}

// getRegistrationDoc retrieves a registration document registered to a specific
// registration ID. Defined as variable to allow for replacement in tests.
var getRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
	_, err := client.Collection(utility.RegistrationsCollection).Doc(id).Get(ctx)
	return err
}

// setRegistrationDoc updates a registration document registered to a specific
// registration ID. Defined as variable to allow for replacement in tests.
var setRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string, reg map[string]any) error {
	_, err := client.Collection(utility.RegistrationsCollection).Doc(id).Set(ctx, reg)
	return err
}

// deleteRegistrationDoc deletes a registration document registered to a specific
// registration ID. Defined as variable to allow for replacement in tests.
var deleteRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
	_, err := client.Collection(utility.RegistrationsCollection).Doc(id).Delete(ctx)
	return err
}

// getRegistrationByIDDocImpl retrieves a registration document by its ID, returns the document data as a map.
// Defined as variable to allow for replacement in tests.
var getRegistrationByIDDocImpl = func(ctx context.Context, client *firestore.Client, id string) (map[string]any, error) {
	doc, err := client.Collection(utility.RegistrationsCollection).Doc(id).Get(ctx)
	if err != nil {
		return nil, err
	}
	return doc.Data(), nil
}

// listRegistrationDocs retrieves all registration documents from Firestore and returns them as a slice of maps.
// Defined as variable to allow for replacement in tests.
var listRegistrationDocs = func(ctx context.Context, client *firestore.Client) ([]map[string]interface{}, error) {
	iter := client.Collection(utility.RegistrationsCollection).Documents(ctx) // Creates an Iterator
	defer iter.Stop()                                                         // Ensures it stops in at the end

	var results []map[string]interface{}
	for { // Document looping
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) { // stop when no more documents
			break
		}
		if err != nil {
			return nil, err
		}
		results = append(results, doc.Data()) // add document data in list
	}
	return results, nil
}

// HandleRegReq routes incoming HTTP requests to the appropriate handler method
// based on the request method. Supports POST for adding registrations and GET
// for retrieving registrations. Responds with 405 Method Not Allowed for unsupported methods.
func (h *Handler) HandleRegReq(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.addRegistration(w, r)
	case http.MethodGet:
		h.handleAllGetRegistration(w, r)
	case http.MethodPut:
		h.replaceRegistration(w, r)
	case http.MethodPatch:
		h.partialUpdateRegistration(w, r)
	case http.MethodDelete:
		h.deleteRegistration(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// addRegistration handles POST requests to the /registrations endpoint.
// It decodes, normalizes and validates the request body, stores a dashboard configuration in
// Firestore, increments the local registration counter and returns the generated
// registration ID and last-change timestamp. Returns 201 Created on success.
func (h *Handler) addRegistration(w http.ResponseWriter, r *http.Request) {
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			log.Printf("Error closing body: %v", err)
		}
	}(r.Body)

	log.Printf("Received %s request", r.Method)

	// Decode, normalize and validate
	regReq, ok := parseRegReq(w, r)
	if !ok {
		return
	}

	ctx := firestoreContext(r)
	lastChange := currentLastChange()

	// Add a registration and return its generated unique ID
	id, err := addRegistrationDocImpl(ctx, h.Client, map[string]any{
		"country":    regReq.Country,
		"isoCode":    regReq.IsoCode,
		"features":   regReq.Features,
		"lastChange": lastChange,
	})
	if err != nil {
		log.Printf("Failed to add registration: %v", err)
		http.Error(w, "failed to add registration", http.StatusInternalServerError)
		return
	}

	log.Printf("Registration created with ID: %s", id)
	// Increase the local registration count by one
	h.RegistrationCount.Add(1)

	w.Header().Set(utility.ContentType, utility.ApplicationJSON)
	w.WriteHeader(http.StatusCreated)

	// Encode the response body as JSON
	encodeRegResp(w, id, lastChange)
}

// handleAllGetRegistration handles GET requests to the /registrations endpoint.
// If the request method is HEAD, delegates to handleHead to return status/headers only
// If no ID is provided in the path, it retrieves all registrations.
// If an ID is provided, it validates the ID length before fetching
// the corresponding registration document from Firestore.
func (h *Handler) handleAllGetRegistration(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s request", r.Method)
	docID := strings.TrimSpace(r.PathValue("id"))
	w.Header().Set(utility.ContentType, utility.ApplicationJSON)

	// HEAD → return headers only
	if r.Method == http.MethodHead {
		h.handleHead(w, r, docID)
		return
	}

	// GET logic
	if docID == "" {
		h.GetAllRegistrations(w, r)
		return
	}

	// If an ISO code is provided, validate it before querying Firestore
	if len(docID) != 20 {
		http.Error(w, "Invalid document ID, the document must be 20 characters", http.StatusBadRequest)
		return
	}

	h.GetRegistrationByID(w, r, docID)
}

// GetAllRegistrations retrieves all registration documents from Firestore and returns them as a
// JSON array in the response body. It iterates through all documents in the "registrations" collection,
// collects their data into a slice of maps, and encodes the result as JSON.
// If an error occurs during retrieval, it responds with a 500 Internal Server Error.
func (h *Handler) GetAllRegistrations(w http.ResponseWriter, r *http.Request) {
	ctx := firestoreContext(r)

	results, err := listRegistrationDocs(ctx, h.Client)
	if err != nil {
		http.Error(w, "Error retrieving data", http.StatusInternalServerError)
		log.Printf("Failed to list registration documents: %v", err)
		return
	}

	_ = json.NewEncoder(w).Encode(results)
}

// handleHead writes only HTTP headers for a HEAD request without returning a body.
// It performs the same validation and lookup logic as the GET handler to determine
// the appropriate status code, but intentionally omits writing any response body,
// as required by the HTTP HEAD method.
// This method mirrors the validation and lookup logic of the GET handler,
// but intentionally omits writing any response body, as required by the HTTP HEAD method.
func (h *Handler) handleHead(w http.ResponseWriter, r *http.Request, docID string) {
	docID = strings.TrimSpace(docID)

	// Checks if the docID is empty, if empty it will return a status code of 200.
	if strings.TrimSpace(docID) == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx := firestoreContext(r)

	// Checks if the document with the provided docID exists in Firestore.
	//If it does not exist, it will return a status code of 404.
	err := getRegistrationDoc(ctx, h.Client, docID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		log.Printf("Document %s does not exist", docID)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetRegistrationByID handles a GET request that retrieves registration data
// for a specific country identified by its document ID. It goes through firebase
// query and check if there is a similar Documents with that ID and returns the
// matching documents as a JSON. if not it returns a status code of 404 not found.
func (h *Handler) GetRegistrationByID(w http.ResponseWriter, r *http.Request, docID string) {
	ctx := firestoreContext(r)

	data, err := getRegistrationByIDDocImpl(ctx, h.Client, docID)
	if err != nil {
		http.Error(w, "Registration not found", http.StatusNotFound)
		log.Printf("Registration with ID %s not found: %v", docID, err)
		return
	}

	_ = json.NewEncoder(w).Encode(data)
}

// replaceRegistration handles PUT requests to the /registrations/{id} endpoint.
// It decodes, normalizes and validates the request body, replaces the stored registration
// configuration for the given ID, updates the last-change timestamp, and returns
// 200 OK with an empty body on success.
func (h *Handler) replaceRegistration(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s request", r.Method)

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		log.Printf("invalid registration id: %q", id)
		http.Error(w, "invalid registration id", http.StatusBadRequest)
		return
	}

	// Decode, normalize and validate
	regReq, ok := parseRegReq(w, r)
	if !ok {
		return
	}

	ctx := firestoreContext(r)
	lastChange := currentLastChange()

	// Ensure registration exists for the provided ID
	errGet := getRegistrationDoc(ctx, h.Client, id)
	if errGet != nil {
		log.Printf("Failed to get registration: %v", errGet)
		http.Error(w, "failed getting registration", http.StatusNotFound)
		return
	}

	// Replaces stored registration for the provided ID
	errSet := setRegistrationDoc(ctx, h.Client, id, map[string]any{
		"country":    regReq.Country,
		"isoCode":    regReq.IsoCode,
		"features":   regReq.Features,
		"lastChange": lastChange,
	})
	if errSet != nil {
		log.Printf("Failed to update registration: %v", errSet)
		http.Error(w, "failed updating registration", http.StatusInternalServerError)
		return
	}

	log.Printf("Replaced registration with ID: %s", id)
	w.WriteHeader(http.StatusOK)
}

// partialUpdateRegistration handles PATCH requests to the /registrations/{id} endpoint.
// It decodes, normalizes and validates the request body, applies partial updates
// of a configuration for the given ID, updates the last-change timestamp and returns
// 200 OK with an empty body on success.
func (h *Handler) partialUpdateRegistration(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s request", r.Method)

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		log.Printf("invalid registration id: %q", id)
		http.Error(w, "invalid registration id", http.StatusBadRequest)
		return
	}

	// Decode, normalize and validate
	regReq, ok := parsePatchRegReq(w, r)
	if !ok {
		return
	}

	ctxPatch := firestoreContext(r)
	lastChange := currentLastChange()

	errGet := getRegistrationDoc(ctxPatch, h.Client, id)
	if errGet != nil {
		log.Printf("Failed to get registration: %v", errGet)
		http.Error(w, "failed getting registration", http.StatusNotFound)
		return
	}

	// Build general PATCH update, append any currency updates
	update := buildGeneralPatchUpdate(&regReq, lastChange)
	update = append(update, buildCurrencyPatchUpdate(regReq.Features)...)

	_, errSet := h.Client.Collection(utility.RegistrationsCollection).Doc(id).Update(ctxPatch, update)
	if errSet != nil {
		log.Printf("Failed to update registration: %v", errSet)
		http.Error(w, "failed updating registration", http.StatusInternalServerError)
		return
	}
	log.Printf("Updated registration with ID: %s", id)
	// Respond with status code 200
	w.WriteHeader(http.StatusOK)
}

// deleteRegistration handles DELETE requests to the /registrations/{id} endpoint.
// Validates the registration ID, ensures the registration exists, then deletes it
// from Firestore.
func (h *Handler) deleteRegistration(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s request", r.Method)

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		log.Printf("invalid registration id: %q", id)
		http.Error(w, "invalid registration id", http.StatusBadRequest)
		return
	}

	ctx := firestoreContext(r)

	// Retrieve registration data to make available for webhook notification
	data, errGet := getRegistrationByIDDocImpl(ctx, h.Client, id)
	if errGet != nil {
		log.Printf("Failed to get registration: %v", errGet)
		http.Error(w, "failed getting registration", http.StatusNotFound)
		return
	}

	isoCode, ok := data["isoCode"].(string)
	if !ok {
		log.Printf("invalid isoCode for registration %q: %#v", id, data["isoCode"])
		http.Error(w, "failed reading registration", http.StatusInternalServerError)
		return
	}

	errDel := deleteRegistrationDoc(ctx, h.Client, id)
	if errDel != nil {
		log.Printf("Failed to delete registration: %v", errDel)
		http.Error(w, "failed deleting registration", http.StatusInternalServerError)
		return
	}
	log.Printf("Deleted registration with ID: %s and isoCode: %s", id, isoCode)
	w.WriteHeader(http.StatusNoContent)
}

// decodeRegReq decodes the JSON request body into a registration request.
// Writes a HTTP error response if decoding fails.
func decodeRegReq(w http.ResponseWriter, r *http.Request, regReq *models.RegistrationRequest) bool {
	if err := json.NewDecoder(r.Body).Decode(regReq); err != nil {
		log.Printf("Failed to decode registration request: %v", err)
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return true
	}
	return false
}

// encodeRegResp encodes a registration response as JSON and logs an error if
// encoding fails.
func encodeRegResp(w http.ResponseWriter, id, lastChange string) {
	if err := json.NewEncoder(w).Encode(models.RegistrationResponse{
		ID:         id,
		LastChange: lastChange,
	}); err != nil {
		log.Printf("Failed to encode registration response: %v", err)
	}
}

// currentLastChange keeps track of time and date of which a registration
// has been created or updated.
func currentLastChange() string {
	lastChange := time.Now().Format(time.RFC3339)
	return lastChange
}

// firestoreContext returns the request context used for Firestore operations.
func firestoreContext(r *http.Request) context.Context {
	return r.Context()
}

// validateRegReq validates registration fields and writes an HTTP error
// response if validation fails.
func validateRegReq(w http.ResponseWriter, regReq models.RegistrationRequest) bool {
	if len(regReq.IsoCode) != utility.IsoCodeLength {
		log.Printf("Invalid isoCode: %q", regReq.IsoCode)
		http.Error(w, "iso-code must be 2-letter country code", http.StatusBadRequest)
		return true
	}

	for _, l := range regReq.IsoCode {
		if !unicode.IsLetter(l) {
			log.Printf("Invalid isoCode: %q", l)
			http.Error(w, "iso-code must contain only letters", http.StatusBadRequest)
			return true
		}
	}

	if regReq.Country == "" {
		log.Printf("invalid country: %q", regReq.Country)
		http.Error(w, "country cannot be blank", http.StatusBadRequest)
		return true
	}

	for _, c := range regReq.Country {
		// Unicode.IsSpace takes into account country names with spaces in the name
		if !unicode.IsLetter(c) && !unicode.IsSpace(c) {
			log.Printf("Invalid country: %q", c)
			http.Error(w, "country must contain only letters", http.StatusBadRequest)
			return true
		}
	}

	for _, f := range regReq.Features.TargetCurrencies {
		if len(f) != utility.CurrencyCodeLength {
			log.Printf("Invalid currency length: %q", f)
			http.Error(w, "target currency must be 3-letter ISO-code", http.StatusBadRequest)
			return true
		}
		for _, l := range f {
			if !unicode.IsLetter(l) {
				log.Printf("Invalid currency code: %q contains invalid character %q", f, l)
				http.Error(w, "target currency must contain only letters", http.StatusBadRequest)
				return true
			}
		}
	}

	return false
}

// normalizeFields normalizes registration request fields into consistent
// format before validation or storage.
func normalizeFields(regReq *models.RegistrationRequest) {
	regReq.Country = strings.TrimSpace(regReq.Country)
	regReq.IsoCode = strings.ToUpper(strings.TrimSpace(regReq.IsoCode))

	// Normalizes each target currency in the slice individually
	for i, c := range regReq.Features.TargetCurrencies {
		regReq.Features.TargetCurrencies[i] = strings.ToUpper(strings.TrimSpace(c))
	}
}

// parseRegReq decodes, normalizes and validates a registration request.
// Returns the parsed request, false if any step fails.
func parseRegReq(w http.ResponseWriter, r *http.Request) (models.RegistrationRequest, bool) {
	var regReq models.RegistrationRequest

	if decodeRegReq(w, r, &regReq) {
		return models.RegistrationRequest{}, false
	}

	normalizeFields(&regReq)

	if validateRegReq(w, regReq) {
		return models.RegistrationRequest{}, false
	}
	return regReq, true
}

// decodePatchRegReq decodes the JSON request body into PATCH registration request.
// Writes an HTTP error response if decoding fails.
func decodePatchRegReq(w http.ResponseWriter, r *http.Request, regReq *models.RegistrationPatchRequest) bool {
	if err := json.NewDecoder(r.Body).Decode(regReq); err != nil {
		log.Printf("Failed to decode patch request: %v", err)
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return true
	}
	return false
}

// normalizePatchFields normalizes registration request fields into consistent
// format before validation or storage.
func normalizePatchFields(regReq *models.RegistrationPatchRequest) {
	if regReq.Country != nil {
		trim := strings.TrimSpace(*regReq.Country)
		regReq.Country = &trim
	}

	if regReq.IsoCode != nil {
		normalized := strings.ToUpper(strings.TrimSpace(*regReq.IsoCode))
		regReq.IsoCode = &normalized
	}

	if regReq.Features != nil {
		// Normalize provided currency lists individually
		normalizeCurrencyList(regReq.Features.TargetCurrencies)
		normalizeCurrencyList(regReq.Features.AddTargetCurrencies)
		normalizeCurrencyList(regReq.Features.RemoveTargetCurrencies)
	}
}

// validatePatchRegReq validates registration fields and writes an HTTP error
// response if validation fails.
func validatePatchRegReq(w http.ResponseWriter, regReq models.RegistrationPatchRequest) bool {
	if regReq.IsoCode != nil {
		if len(*regReq.IsoCode) != utility.IsoCodeLength {
			log.Printf("Invalid isoCode: %q", *regReq.IsoCode)
			http.Error(w, "iso-code must be 2-letter country code", http.StatusBadRequest)
			return true
		}
		for _, l := range *regReq.IsoCode {
			if !unicode.IsLetter(l) {
				log.Printf("Invalid isoCode: %q", l)
				http.Error(w, "iso-code must contain only letters", http.StatusBadRequest)
				return true
			}
		}
	}

	if regReq.Country != nil {
		if *regReq.Country == "" {
			log.Printf("invalid country: %q", *regReq.Country)
			http.Error(w, "country cannot be blank", http.StatusBadRequest)
			return true
		}
		for _, c := range *regReq.Country {
			// Unicode.IsSpace takes into account country names with spaces in the name
			if !unicode.IsLetter(c) && !unicode.IsSpace(c) {
				log.Printf("Invalid country: %q", c)
				http.Error(w, "country must contain only letters", http.StatusBadRequest)
				return true
			}
		}
	}

	// Currency PATCH modes (replace, add, remove) cannot be combined in the same request
	if regReq.Features != nil {
		f := regReq.Features

		if f.TargetCurrencies != nil && (f.AddTargetCurrencies != nil || f.RemoveTargetCurrencies != nil) {
			log.Printf("Invalid target currencies: %q", f.TargetCurrencies)
			http.Error(w, "target currencies cannot be replaced and added/removed in the same request", http.StatusBadRequest)
			return true
		}

		if f.AddTargetCurrencies != nil && f.RemoveTargetCurrencies != nil {
			log.Printf("Invalid target currencies: %q", f.RemoveTargetCurrencies)
			http.Error(w, "target currencies cannot be added and removed in the same request", http.StatusBadRequest)
			return true
		}

		if validateCurrencyList(w, f.TargetCurrencies, utility.TargetCurrency) {
			return true
		}

		if validateCurrencyList(w, f.AddTargetCurrencies, utility.AddTargetCurrency) {
			return true
		}

		if validateCurrencyList(w, f.RemoveTargetCurrencies, utility.RemoveTargetCurrency) {
			return true
		}
	}

	return false
}

// parsePatchRegReq decodes, normalizes and validates a registration request.
// Returns the parsed request, false if any step fails.
func parsePatchRegReq(w http.ResponseWriter, r *http.Request) (models.RegistrationPatchRequest, bool) {
	var regReq models.RegistrationPatchRequest

	if decodePatchRegReq(w, r, &regReq) {
		return models.RegistrationPatchRequest{}, false
	}

	normalizePatchFields(&regReq)

	if validatePatchRegReq(w, regReq) {
		return models.RegistrationPatchRequest{}, false
	}
	return regReq, true
}

// normalizeCurrencyList normalizes each currency code in the slice.
func normalizeCurrencyList(currencies *[]string) {
	if currencies == nil {
		return
	}

	for i, c := range *currencies {
		(*currencies)[i] = strings.ToUpper(strings.TrimSpace(c))
	}
}

// validateCurrencyList validates a list of currency codes.
// Writes an HTTP error response if validation fails.
func validateCurrencyList(w http.ResponseWriter, currencies *[]string, fieldName string) bool {
	if currencies == nil {
		return false
	}

	for _, curr := range *currencies {
		if len(curr) != utility.CurrencyCodeLength {
			log.Printf("Invalid currency length: %q", curr)
			http.Error(w, fieldName+" must be 3-letter ISO-code", http.StatusBadRequest)
			return true
		}
		for _, l := range curr {
			if !unicode.IsLetter(l) {
				log.Printf("Invalid currency code: %q contains invalid character %q", curr, l)
				http.Error(w, fieldName+" must contain only letters", http.StatusBadRequest)
				return true
			}
		}
	}
	return false
}

// boolUpdateField is a Firestore update path with an
// optional boolean value to include in PATCH update.
type boolUpdateField struct {
	path string
	val  *bool
}

// addPatchStringUpdateIfSet appends a Firestore string update if
// value is not nil.
func addPatchStringUpdateIfSet(update []firestore.Update, path string, val *string) []firestore.Update {
	if val != nil {
		update = append(update, firestore.Update{
			Path:  path,
			Value: *val,
		})
	}
	return update
}

// addPatchBoolUpdateIfSet appends Firestore bool updates for all
// fields whose values are not nil.
func addPatchBoolUpdateIfSet(update []firestore.Update, fields []boolUpdateField) []firestore.Update {
	for _, f := range fields {
		if f.val != nil {
			update = append(update, firestore.Update{
				Path:  f.path,
				Value: *f.val,
			})
		}
	}
	return update
}

// buildGeneralPatchUpdate creates Firestore update for general PATCH
// fields. Includes an updated last-change timestamp.
func buildGeneralPatchUpdate(patch *models.RegistrationPatchRequest, lastChange string) []firestore.Update {
	update := []firestore.Update{
		{Path: "lastChange", Value: lastChange},
	}

	update = addPatchStringUpdateIfSet(update, "country", patch.Country)
	update = addPatchStringUpdateIfSet(update, "isoCode", patch.IsoCode)

	// Add feature updates only for fields included in the PATCH request.
	if patch.Features != nil {
		f := patch.Features

		update = addPatchBoolUpdateIfSet(update, []boolUpdateField{
			{"features.temperature", f.Temperature},
			{"features.precipitation", f.Precipitation},
			{"features.airQuality", f.AirQuality},
			{"features.capital", f.Capital},
			{"features.coordinates", f.Coordinates},
			{"features.population", f.Population},
			{"features.area", f.Area},
		})
	}
	return update
}

// buildReplaceCurrencyUpdate creates a Firestore update slice for replacing
// all targetCurrencies fields with the provided currency list.
func buildReplaceCurrencyUpdate(currencies *[]string) []firestore.Update {
	if currencies == nil {
		return nil
	}

	return []firestore.Update{
		{
			Path:  "features.targetCurrencies",
			Value: *currencies,
		},
	}
}

// buildAddCurrencyUpdate creates a Firestore update slice for adding
// currencies to the targetCurrencies list.
func buildAddCurrencyUpdate(currencies *[]string) []firestore.Update {
	if currencies == nil || len(*currencies) == 0 {
		return nil
	}

	// Convert currency slice into []any so it can be passed to firestore operation
	values := make([]any, len(*currencies))
	for i, c := range *currencies {
		values[i] = c
	}

	return []firestore.Update{
		{
			Path: "features.targetCurrencies",
			// Uses values... in order to pass each currency as a separate argument
			Value: firestore.ArrayUnion(values...),
		},
	}
}

// buildRemoveCurrencyUpdate creates a Firestore update slice for removing
// currencies from the targetCurrencies list.
func buildRemoveCurrencyUpdate(currencies *[]string) []firestore.Update {
	if currencies == nil || len(*currencies) == 0 {
		return nil
	}

	// Convert currency slice into []any so it can be passed to firestore operation
	values := make([]any, len(*currencies))
	for i, c := range *currencies {
		values[i] = c
	}

	return []firestore.Update{
		{
			Path: "features.targetCurrencies",
			// Uses values... in order to pass each currency as a separate argument
			Value: firestore.ArrayRemove(values...),
		},
	}
}

// buildCurrencyPatchUpdate constructs a Firestore update slice for PATCH requests
// that can replace, add or remove target currencies.
func buildCurrencyPatchUpdate(features *models.RegistrationPatchFeatures) []firestore.Update {
	// Return early to avoid dereferencing a nil feature pointer
	if features == nil {
		return nil
	}

	switch {
	case features.TargetCurrencies != nil:
		return buildReplaceCurrencyUpdate(features.TargetCurrencies)
	case features.AddTargetCurrencies != nil:
		return buildAddCurrencyUpdate(features.AddTargetCurrencies)
	case features.RemoveTargetCurrencies != nil:
		return buildRemoveCurrencyUpdate(features.RemoveTargetCurrencies)
	default:
		return nil
	}
}
