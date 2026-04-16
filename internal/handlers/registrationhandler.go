package handlers

import (
	"assignment-2/internal/clients"
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

// Handler holds shared dependencies, local registration
// and webhook counter, and API client interface.
type Handler struct {
	Client                          *firestore.Client
	RegistrationCount, WebhookCount atomic.Int64
	API                             clients.APIClient
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

// getRegistrationDoc checks whether a registration document exists
// for the given registration ID. Defined as variable to allow
// for replacement in tests.
var getRegistrationDoc = func(ctx context.Context, client *firestore.Client, id string) error {
	_, err := client.Collection(utility.RegistrationsCollection).Doc(id).Get(ctx)
	return err
}

// setRegistrationDocsets the registration document for the given
// registration ID. Defined as variable to allow replacement in tests
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

// getRegistrationByIDFunc retrieves a registration by document ID.
// Defined as variable to allow replacement in tests.
var getRegistrationByIDFunc = getRegistrationByID

// HandleRegReq routes incoming HTTP requests to the appropriate handler method
// based on the request method. Supports POST, GET, HEAD, PUT, PATCH and DELETE
// for registration resources. Responds with 405 Method Not Allowed for unsupported
// methods.
func (h *Handler) HandleRegReq(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.addRegistration(w, r)
	case http.MethodGet, http.MethodHead:
		h.handleAllGetRegistration(w, r)
	case http.MethodPut:
		h.replaceRegistration(w, r)
	case http.MethodPatch:
		h.partialUpdateRegistration(w, r)
	case http.MethodDelete:
		h.deleteRegistration(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// addRegistration handles POST requests to the /registrations endpoint.
// It decodes, normalizes, and validates the request body, resolves a missing
// country or isoCode when only one is provided, stores the registration
// configuration in Firestore, increments the local registration counter.
// Triggers the REGISTER lifecycle webhook and returns 201 Created with the
// generated registration ID and last-change timestamp on success.
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

	country, isoCode, done := resolveRegReqIdentity(w, ctx, regReq)
	if done {
		return
	}

	// Add a registration and return its generated unique ID
	id, err := addRegistrationDocImpl(ctx, h.Client, map[string]any{
		"country":    country,
		"isoCode":    isoCode,
		"features":   regReq.Features,
		"lastChange": lastChange,
	})
	if err != nil {
		log.Printf("Failed to add registration: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to add registration")
		return
	}

	log.Printf("Registration created with ID: %s", id)
	// Increase the local registration count by one
	h.RegistrationCount.Add(1)
	h.triggerLifecycleWebhooks(ctx, "REGISTER", isoCode)

	w.Header().Set(utility.ContentType, utility.ApplicationJSON)
	w.WriteHeader(http.StatusCreated)

	// Encode the response body as JSON
	encodeRegResp(w, id, lastChange)
}

// handleAllGetRegistration handles GET and HEAD requests to the /registrations
// and /registrations/{id} endpoints.
// If no ID is provided, GET returns all registrations while HEAD returns 200 OK
// with no response body.
// If an ID is provided, validates the document ID length. For a valid ID, HEAD
// checks whether the registration exists and returns only the appropriate status code,
// while GET returns the corresponding registration document as JSON.
func (h *Handler) handleAllGetRegistration(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s request", r.Method)

	docID := strings.TrimSpace(r.PathValue("id"))
	w.Header().Set(utility.ContentType, utility.ApplicationJSON)

	// GET logic
	if docID == "" {
		// HEAD -> return headers only
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.GetAllRegistrations(w, r)
		return
	}

	// If a document ID is provided, validate it before querying Firestore
	if len(docID) != utility.RegistrationDocIDLength {
		// HEAD -> return headers only
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		writeJSONError(w, http.StatusBadRequest, "Invalid document ID, the document must be 20 characters")
		return
	}

	// HEAD -> return headers only
	if r.Method == http.MethodHead {
		h.handleHead(w, r, docID)
		return
	}

	h.GetRegistrationByID(w, r, docID)
}

// GetAllRegistrations retrieves all registration documents from Firestore and returns them as a
// JSON array in the response body. It iterates through all documents in the "registrations" collection,
// converts each document into a StoredRegistration struct, and encodes the result as JSON.
// If an error occurs during retrieval or decoding, it responds with HTTP 500 Internal Server Error.
func (h *Handler) GetAllRegistrations(w http.ResponseWriter, r *http.Request) {
	ctx := firestoreContext(r)

	iter := h.Client.Collection(utility.RegistrationsCollection).Documents(ctx)
	defer iter.Stop()

	var results []models.StoredRegistration

	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Error retrieving data")
			log.Printf("Failed to list registration documents: %v", err)
			return
		}

		var reg models.StoredRegistration
		if err := doc.DataTo(&reg); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to decode the registration document")
			log.Printf("Failed to decode document: %v", err)
			return
		}

		reg.ID = doc.Ref.ID

		results = append(results, reg)
	}

	w.Header().Set(utility.ContentType, utility.ApplicationJSON)
	_ = json.NewEncoder(w).Encode(results)
}

// handleHead handles HEAD requests for /registrations/{id} endpoint.
// It checks whether the requested registration exists, writes only the
// appropriate HTTP status code, without returning a response body.
// Returns 200 OK if the registration exists, or 404 Not Found otherwise.
func (h *Handler) handleHead(w http.ResponseWriter, r *http.Request, docID string) {
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

// GetRegistrationByID handles a GET request that retrieves a single registration
// document by its Firestore document ID. If the document exists, it is returned
// as JSON. If the document is not found, handler responds with 404 Not Found.
// If the stored document cannot be decoded, responds with 500 Internal Server Error.
func (h *Handler) GetRegistrationByID(w http.ResponseWriter, r *http.Request, docID string) {
	ctx := firestoreContext(r)

	doc, err := h.Client.Collection(utility.RegistrationsCollection).Doc(docID).Get(ctx)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "The document was not found")
		log.Printf("Registration with ID %s not found: %v", docID, err)
		return
	}

	var reg models.StoredRegistration
	if err := doc.DataTo(&reg); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to decode the registration document")
		log.Printf("Failed to decode document: %v", err)
		return
	}

	reg.ID = doc.Ref.ID

	w.Header().Set(utility.ContentType, utility.ApplicationJSON)
	_ = json.NewEncoder(w).Encode(reg)
}

// replaceRegistration handles PUT requests to the /registrations/{id} endpoint.
// It decodes, normalizes, and validates the request body, ensures the registration exists,
// resolves a missing country or isoCode when only one is provided, replaces the stored
// registration for the given ID, updates the last-change timestamp. Triggers the CHANGE
// lifecycle webhook and returns 200 OK with an empty body on success.
func (h *Handler) replaceRegistration(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s request", r.Method)

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		log.Printf("invalid registration id: %q", id)
		writeJSONError(w, http.StatusBadRequest, "invalid registration id")
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
	_, errGet := getRegistrationByIDFunc(ctx, h.Client, id)
	if errGet != nil {
		log.Printf("Failed to get registration: %v", errGet)
		writeJSONError(w, http.StatusNotFound, "failed getting registration")
		return
	}

	country, isoCode, done := resolveRegReqIdentity(w, ctx, regReq)
	if done {
		return
	}
	// Replaces stored registration for the provided ID
	errSet := setRegistrationDoc(ctx, h.Client, id, map[string]any{
		"country":    country,
		"isoCode":    isoCode,
		"features":   regReq.Features,
		"lastChange": lastChange,
	})
	if errSet != nil {
		log.Printf("Failed to update registration: %v", errSet)
		writeJSONError(w, http.StatusInternalServerError, "failed updating registration")
		return
	}

	h.triggerLifecycleWebhooks(ctx, "CHANGE", isoCode)

	log.Printf("Replaced registration with ID: %s", id)
	w.WriteHeader(http.StatusOK)
}

// partialUpdateRegistration handles PATCH requests to the /registrations/{id} endpoint.
// It decodes, normalizes and validates the request body, ensures registration exists,
// resolves a missing country or isoCode if only one is provided, applies partial updates
// to the stored registration, updates the last-change timestamp. Triggers the CHANGE webhook
// lifecycle and returns 200 OK with an empty body on success.
func (h *Handler) partialUpdateRegistration(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s request", r.Method)

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		log.Printf("invalid registration id: %q", id)
		writeJSONError(w, http.StatusBadRequest, "invalid registration id")
		return
	}

	// Decode, normalize and validate
	regReq, ok := parsePatchRegReq(w, r)
	if !ok {
		return
	}

	ctx := firestoreContext(r)
	lastChange := currentLastChange()

	reg, errGet := getRegistrationByIDFunc(ctx, h.Client, id)
	if errGet != nil {
		log.Printf("Failed to get registration: %v", errGet)
		writeJSONError(w, http.StatusNotFound, "failed getting registration")
		return
	}

	if err := resolvePatchIdentity(ctx, &regReq); err != nil {
		log.Printf("Failed to resolve patch identity: %v", err)
		writeJSONError(w, http.StatusBadRequest, "failed resolving country or iso-code")
		return
	}

	// Build general PATCH update, append any currency updates
	update := buildGeneralPatchUpdate(&regReq, lastChange)
	update = append(update, buildCurrencyPatchUpdate(regReq.Features)...)

	_, errSet := h.Client.Collection(utility.RegistrationsCollection).Doc(id).Update(ctx, update)
	if errSet != nil {
		log.Printf("Failed to update registration: %v", errSet)
		writeJSONError(w, http.StatusInternalServerError, "failed updating registration")
		return
	}

	isoCode := reg.IsoCode
	if regReq.IsoCode != nil {
		isoCode = *regReq.IsoCode
	}

	h.triggerLifecycleWebhooks(ctx, "CHANGE", isoCode)
	log.Printf("Updated registration with ID: %s", id)
	// Respond with status code 200
	w.WriteHeader(http.StatusOK)
}

// deleteRegistration handles DELETE requests to the /registrations/{id} endpoint.
// Validates the registration ID, ensures the registration exists, then deletes it
// from Firestore. Triggers a DELETE lifecycle webhook, and returns 204 No Content
// on success.
func (h *Handler) deleteRegistration(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s request", r.Method)

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		log.Printf("invalid registration id: %q", id)
		writeJSONError(w, http.StatusBadRequest, "invalid registration id")
		return
	}

	ctx := firestoreContext(r)

	// Retrieve registration reg to make available for webhook notification
	reg, errGet := getRegistrationByIDFunc(ctx, h.Client, id)
	if errGet != nil {
		log.Printf("Failed to get registration: %v", errGet)
		writeJSONError(w, http.StatusNotFound, "failed getting registration")
		return
	}

	if reg.IsoCode == "" {
		log.Printf("missing isoCode for registration %q", id)
		writeJSONError(w, http.StatusInternalServerError, "failed reading registration")
		return
	}

	errDel := deleteRegistrationDoc(ctx, h.Client, id)
	if errDel != nil {
		log.Printf("Failed to delete registration: %v", errDel)
		writeJSONError(w, http.StatusInternalServerError, "failed deleting registration")
		return
	}
	h.triggerLifecycleWebhooks(ctx, "DELETE", reg.IsoCode)
	log.Printf("Deleted registration with ID: %s and isoCode: %s", id, reg.IsoCode)
	w.WriteHeader(http.StatusNoContent)
}

// decodeRegReq decodes the JSON request body into a registration request.
// Writes a HTTP error response if decoding fails.
func decodeRegReq(w http.ResponseWriter, r *http.Request, regReq *models.RegistrationRequest) bool {
	if err := json.NewDecoder(r.Body).Decode(regReq); err != nil {
		log.Printf("Failed to decode registration request: %v", err)
		writeJSONError(w, http.StatusBadRequest, "invalid json payload")
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

// currentLastChange returns the current timestamp in RFC3339 format
// for a created or updated registration.
func currentLastChange() string {
	lastChange := time.Now().Format(time.RFC3339)
	return lastChange
}

// firestoreContext returns the request context used for Firestore operations.
func firestoreContext(r *http.Request) context.Context {
	return r.Context()
}

// resolveRegReqIdentity resolves a registration request into a country name
// and ISO code pair. If it fails, writes a JSON error response and returns
// failure flag as true.
func resolveRegReqIdentity(w http.ResponseWriter, ctx context.Context, regReq models.RegistrationRequest) (string, string, bool) {
	country, isoCode, errResolve := resolveRegistrationIdentity(ctx, regReq)
	if errResolve != nil {
		log.Printf("Failed to resolve registration identity: %v", errResolve)
		writeJSONError(w, http.StatusBadRequest, "failed resolving country or iso-code")
		return "", "", true
	}
	return country, isoCode, false
}

// resolveRegistrationIdentity resolves a registration request into a complete
// country name and ISO code pair. If only ISO Code or country name is provided,
// looks up the missing field using the REST Countries client.
func resolveRegistrationIdentity(ctx context.Context, regReq models.RegistrationRequest) (string, string, error) {
	country := regReq.Country
	isoCode := regReq.IsoCode

	switch {
	case country != "" && isoCode != "":
		return country, isoCode, nil

	case isoCode != "":
		info, err := clients.FetchCountryInfoFunc(ctx, isoCode)
		if err != nil {
			return "", "", err
		}
		return info.Name.Common, isoCode, nil

	case country != "":
		info, err := clients.FetchCountryByNameFunc(ctx, country)
		if err != nil {
			return "", "", err
		}
		return info.Name.Common, info.ISOCode, nil

	default:
		return "", "", errors.New("country or isoCode must be provided")
	}
}

// validateRegReq validates registration fields and writes an HTTP error
// response if validation fails.
func validateRegReq(w http.ResponseWriter, regReq models.RegistrationRequest) bool {
	if regReq.Country == "" && regReq.IsoCode == "" {
		log.Printf("invalid registration request: both country and isoCode are blank")
		writeJSONError(w, http.StatusBadRequest, "country or iso-code must be provided")
		return true
	}

	if regReq.IsoCode != "" {
		if len(regReq.IsoCode) != utility.IsoCodeLength {
			log.Printf("invalid isoCode: %q", regReq.IsoCode)
			writeJSONError(w, http.StatusBadRequest, "iso-code must be 2-letter country code")
			return true
		}

		for _, l := range regReq.IsoCode {
			if !unicode.IsLetter(l) {
				log.Printf("Invalid isoCode: %q", l)
				writeJSONError(w, http.StatusBadRequest, "iso-code must contain only letters")
				return true
			}
		}
	}

	if regReq.Country != "" {
		for _, c := range regReq.Country {
			// Unicode.IsSpace takes into account country names with spaces in the name
			if !unicode.IsLetter(c) && !unicode.IsSpace(c) &&
				c != '-' && // allow countries which use certain symbols in their name
				c != '\'' &&
				c != '’' {
				log.Printf("Invalid country: %q", c)
				writeJSONError(w, http.StatusBadRequest, "country must contain only letters")
				return true
			}
		}
	}

	for _, f := range regReq.Features.TargetCurrencies {
		if len(f) != utility.CurrencyCodeLength {
			log.Printf("Invalid currency length: %q", f)
			writeJSONError(w, http.StatusBadRequest, "target currency must be 3-letter ISO-code")
			return true
		}
		for _, l := range f {
			if !unicode.IsLetter(l) {
				log.Printf("Invalid currency code: %q contains invalid character %q", f, l)
				writeJSONError(w, http.StatusBadRequest, "target currency must contain only letters")
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

// resolvePatchIdentity resolves a missing country or isoCode in a PATCH request.
// If neither field is provided, or both are already provided, it leaves the
// request unchanged. If only isoCode is provided, it resolves and sets the
// corresponding country. If only country is provided, it resolves and sets both
// the normalized country name and corresponding isoCode. It returns an error if
// the lookup fails.
func resolvePatchIdentity(ctx context.Context, patch *models.RegistrationPatchRequest) error {
	switch {
	case patch.Country == nil && patch.IsoCode == nil:
		return nil

	case patch.Country != nil && patch.IsoCode != nil:
		return nil

	case patch.IsoCode != nil:
		info, err := clients.FetchCountryInfoFunc(ctx, *patch.IsoCode)
		if err != nil {
			return err
		}

		country := info.Name.Common
		patch.Country = &country
		return nil

	default:
		info, err := clients.FetchCountryByNameFunc(ctx, *patch.Country)
		if err != nil {
			return err
		}

		country := info.Name.Common
		isoCode := info.ISOCode

		patch.Country = &country
		patch.IsoCode = &isoCode
		return nil
	}
}

// decodePatchRegReq decodes the JSON request body into PATCH registration request.
// Writes an HTTP error response if decoding fails.
func decodePatchRegReq(w http.ResponseWriter, r *http.Request, regReq *models.RegistrationPatchRequest) bool {
	if err := json.NewDecoder(r.Body).Decode(regReq); err != nil {
		log.Printf("Failed to decode patch request: %v", err)
		writeJSONError(w, http.StatusBadRequest, "invalid json payload")
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
			writeJSONError(w, http.StatusBadRequest, "iso-code must be 2-letter country code")
			return true
		}
		for _, l := range *regReq.IsoCode {
			if !unicode.IsLetter(l) {
				log.Printf("Invalid isoCode: %q", l)
				writeJSONError(w, http.StatusBadRequest, "iso-code must contain only letters")
				return true
			}
		}
	}

	if regReq.Country != nil {
		if *regReq.Country == "" {
			log.Printf("invalid country: %q", *regReq.Country)
			writeJSONError(w, http.StatusBadRequest, "country cannot be blank")
			return true
		}
		for _, c := range *regReq.Country {
			// Unicode.IsSpace takes into account country names with spaces in the name
			if !unicode.IsLetter(c) && !unicode.IsSpace(c) &&
				c != '-' && // allow countries which use certain symbols in the name
				c != '\'' &&
				c != '’' {
				log.Printf("Invalid country: %q", c)
				writeJSONError(w, http.StatusBadRequest, "country must contain only letters")
				return true
			}
		}
	}

	// Currency PATCH modes (replace, add, remove) cannot be combined in the same request
	if regReq.Features != nil {
		f := regReq.Features

		if f.TargetCurrencies != nil && (f.AddTargetCurrencies != nil || f.RemoveTargetCurrencies != nil) {
			log.Printf("Invalid target currencies: %q", f.TargetCurrencies)
			writeJSONError(w, http.StatusBadRequest, "target currencies cannot be replaced and added/removed in the same request")
			return true
		}

		if f.AddTargetCurrencies != nil && f.RemoveTargetCurrencies != nil {
			log.Printf("Invalid target currencies: %q", f.RemoveTargetCurrencies)
			writeJSONError(w, http.StatusBadRequest, "target currencies cannot be added and removed in the same request")
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

// parsePatchRegReq decodes, normalizes, and validates a PATCH
// registration request. Returns the parsed request and false if
// any step fails.
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
			writeJSONError(w, http.StatusBadRequest, fieldName+" must be 3-letter ISO-code")
			return true
		}
		for _, l := range curr {
			if !unicode.IsLetter(l) {
				log.Printf("Invalid currency code: %q contains invalid character %q", curr, l)
				writeJSONError(w, http.StatusBadRequest, fieldName+" must contain only letters")
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
